package tests_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	clientconfig "github.com/amalshaji/portr/internal/client/config"
	clientdb "github.com/amalshaji/portr/internal/client/db"
	clienttunnel "github.com/amalshaji/portr/internal/client/tunnel"
	"github.com/amalshaji/portr/internal/constants"
	serverconfig "github.com/amalshaji/portr/internal/server/config"
	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/amalshaji/portr/internal/server/wstunnel"
	"github.com/glebarez/sqlite"
	"golang.org/x/net/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testConnectionID = "ci-data-flow-connection"
	testSecretKey    = "ci-data-flow-secret"
	testSubdomain    = "ci-data-flow"
	testTimeout      = 10 * time.Second
)

type tunnelHarness struct {
	client       *clienttunnel.Client
	cancel       context.CancelFunc
	clientErr    chan error
	publicServer *httptest.Server
	publicHost   string
}

func openTestDatabase(t *testing.T, name string, models ...any) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), name+".sqlite")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open %s database: %v", name, err)
	}
	if err := database.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate %s database: %v", name, err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get %s sql database: %v", name, err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return database
}

func backendAddress(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse backend URL: %v", err)
	}
	host, portText, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split backend address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse backend port: %v", err)
	}
	return host, port
}

func startTunnelHarness(t *testing.T, backendURL string) *tunnelHarness {
	t.Helper()

	serverDatabase := openTestDatabase(t, "server", &serverdb.TeamUser{}, &serverdb.Connection{})
	teamUser := serverdb.TeamUser{SecretKey: testSecretKey, Role: "member"}
	if err := serverDatabase.Create(&teamUser).Error; err != nil {
		t.Fatalf("create tunnel user: %v", err)
	}
	subdomain := testSubdomain
	connection := serverdb.Connection{
		ID:          testConnectionID,
		Type:        string(constants.Http),
		Subdomain:   &subdomain,
		Status:      "reserved",
		CreatedByID: teamUser.ID,
	}
	if err := serverDatabase.Create(&connection).Error; err != nil {
		t.Fatalf("create reserved connection: %v", err)
	}

	serverConfig := &serverconfig.Config{
		Proxy:        serverconfig.ProxyConfig{Host: "localhost"},
		Domain:       "localhost",
		UseLocalHost: true,
		Debug:        true,
	}
	serverService := service.New(&serverdb.Db{Conn: serverDatabase})
	proxyServer := proxy.New(serverConfig)
	proxyServer.SetTunnelManager(wstunnel.New(serverConfig, serverService))
	publicServer := httptest.NewServer(proxyServer)
	publicURL, err := url.Parse(publicServer.URL)
	if err != nil {
		publicServer.Close()
		t.Fatalf("parse public server URL: %v", err)
	}
	_, portText, err := net.SplitHostPort(publicURL.Host)
	if err != nil {
		publicServer.Close()
		t.Fatalf("split public server address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		publicServer.Close()
		t.Fatalf("parse public server port: %v", err)
	}
	serverConfig.Proxy.Port = port
	publicHost := net.JoinHostPort("localhost", portText)

	clientDatabase := openTestDatabase(
		t,
		"client",
		&clientdb.Request{},
		&clientdb.WebSocketSession{},
		&clientdb.WebSocketEvent{},
	)
	backendHost, backendPort := backendAddress(t, backendURL)
	client := clienttunnel.New(clientconfig.ClientConfig{
		ServerUrl:              publicHost,
		WsUrl:                  publicHost,
		TunnelUrl:              publicHost,
		SecretKey:              testSecretKey,
		ConnectionID:           testConnectionID,
		UseLocalHost:           true,
		Debug:                  true,
		HealthCheckInterval:    60,
		HealthCheckMaxRetries:  1,
		DisableTerminalLogs:    true,
		EnableRequestLogging:   true,
		EnableHttpReverseProxy: true,
		RedactHeaders:          append([]string(nil), clientconfig.DefaultRedactHeaders...),
		Tunnel: clientconfig.Tunnel{
			Name:      "ci-data-flow",
			Subdomain: testSubdomain,
			Host:      backendHost,
			Port:      backendPort,
			Type:      constants.Http,
		},
	}, &clientdb.Db{Conn: clientDatabase}, nil, nil)

	started := make(chan struct{}, 1)
	client.SetEventHandler(func(event clienttunnel.Event) {
		if event.Type == clienttunnel.EventStarted {
			select {
			case started <- struct{}{}:
			default:
			}
		}
	})
	clientContext, cancel := context.WithCancel(context.Background())
	clientErr := make(chan error, 1)
	go func() {
		clientErr <- client.Start(clientContext)
	}()

	select {
	case <-started:
	case err := <-clientErr:
		cancel()
		publicServer.Close()
		t.Fatalf("tunnel client stopped before becoming ready: %v", err)
	case <-time.After(testTimeout):
		cancel()
		publicServer.Close()
		t.Fatal("timed out waiting for tunnel client readiness")
	}

	harness := &tunnelHarness{
		client:       client,
		cancel:       cancel,
		clientErr:    clientErr,
		publicServer: publicServer,
		publicHost:   publicHost,
	}
	t.Cleanup(func() { harness.close(t) })
	return harness
}

func (h *tunnelHarness) publicTunnelHost() string {
	return testSubdomain + "." + h.publicHost
}

func (h *tunnelHarness) close(t *testing.T) {
	t.Helper()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), testTimeout)
	defer cancelShutdown()
	if err := h.client.Shutdown(shutdownContext); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!strings.Contains(strings.ToLower(err.Error()), "closed network connection") {
		t.Errorf("shut down tunnel client: %v", err)
	}
	h.cancel()
	select {
	case err := <-h.clientErr:
		if err != nil {
			t.Errorf("tunnel client exited with error: %v", err)
		}
	case <-time.After(testTimeout):
		t.Error("timed out waiting for tunnel client shutdown")
	}
	h.publicServer.Close()
}

func TestTunnelDataFlowHTTP(t *testing.T) {
	releaseResponse := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseResponse) }) }
	t.Cleanup(release)

	requestSeen := make(chan error, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err == nil && string(body) != "request-through-tunnel" {
			err = fmt.Errorf("unexpected request body %q", body)
		}
		if err == nil && r.Header.Get("X-Data-Flow") != "ci" {
			err = fmt.Errorf("request header was not preserved")
		}
		requestSeen <- err

		w.Header().Set("Trailer", "X-Tunnel-Trailer")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "first-chunk")
		w.(http.Flusher).Flush()
		select {
		case <-releaseResponse:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, "second-chunk")
		w.Header().Set("X-Tunnel-Trailer", "complete")
	}))
	defer backend.Close()

	harness := startTunnelHarness(t, backend.URL)
	request, err := http.NewRequest(http.MethodPost, harness.publicServer.URL+"/stream?source=ci", strings.NewReader("request-through-tunnel"))
	if err != nil {
		t.Fatalf("create public request: %v", err)
	}
	request.Host = harness.publicTunnelHost()
	request.Header.Set("X-Data-Flow", "ci")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		release()
		t.Fatalf("send request through tunnel: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		release()
		t.Fatalf("unexpected response status %d", response.StatusCode)
	}
	select {
	case err := <-requestSeen:
		if err != nil {
			release()
			t.Fatal(err)
		}
	case <-time.After(testTimeout):
		release()
		t.Fatal("local backend did not receive tunneled request")
	}

	firstChunk := make([]byte, len("first-chunk"))
	firstRead := make(chan error, 1)
	go func() {
		_, readErr := io.ReadFull(response.Body, firstChunk)
		firstRead <- readErr
	}()
	select {
	case err := <-firstRead:
		if err != nil {
			release()
			t.Fatalf("read first streamed response chunk: %v", err)
		}
	case <-time.After(testTimeout):
		release()
		t.Fatal("response was buffered instead of streamed through the tunnel")
	}
	if string(firstChunk) != "first-chunk" {
		release()
		t.Fatalf("unexpected first response chunk %q", firstChunk)
	}

	release()
	rest, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read final response chunk: %v", err)
	}
	if string(rest) != "second-chunk" {
		t.Fatalf("unexpected final response chunk %q", rest)
	}
	if response.Trailer.Get("X-Tunnel-Trailer") != "complete" {
		t.Fatalf("response trailer was not preserved: %q", response.Trailer.Get("X-Tunnel-Trailer"))
	}
}

func TestTunnelDataFlowWebSocket(t *testing.T) {
	backendMessage := make(chan string, 1)
	backendError := make(chan error, 1)
	backend := httptest.NewServer(websocket.Handler(func(connection *websocket.Conn) {
		var message string
		if err := websocket.Message.Receive(connection, &message); err != nil {
			backendError <- err
			return
		}
		backendMessage <- message
		backendError <- websocket.Message.Send(connection, "echo:"+message)
	}))
	defer backend.Close()

	harness := startTunnelHarness(t, backend.URL)
	publicURL, err := url.Parse(harness.publicServer.URL)
	if err != nil {
		t.Fatalf("parse public proxy URL: %v", err)
	}
	rawConnection, err := net.DialTimeout("tcp", publicURL.Host, testTimeout)
	if err != nil {
		t.Fatalf("dial public proxy: %v", err)
	}
	websocketConfig, err := websocket.NewConfig("ws://"+harness.publicTunnelHost()+"/echo", "http://"+harness.publicTunnelHost())
	if err != nil {
		_ = rawConnection.Close()
		t.Fatalf("create websocket config: %v", err)
	}
	connection, err := websocket.NewClient(websocketConfig, rawConnection)
	if err != nil {
		_ = rawConnection.Close()
		t.Fatalf("upgrade websocket through tunnel: %v", err)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(testTimeout))

	if err := websocket.Message.Send(connection, "websocket-through-tunnel"); err != nil {
		t.Fatalf("send websocket frame through tunnel: %v", err)
	}
	var response string
	if err := websocket.Message.Receive(connection, &response); err != nil {
		t.Fatalf("receive websocket frame through tunnel: %v", err)
	}
	if response != "echo:websocket-through-tunnel" {
		t.Fatalf("unexpected websocket response %q", response)
	}
	select {
	case message := <-backendMessage:
		if message != "websocket-through-tunnel" {
			t.Fatalf("local backend received unexpected websocket message %q", message)
		}
	case <-time.After(testTimeout):
		t.Fatal("local backend did not receive websocket frame")
	}
	select {
	case err := <-backendError:
		if err != nil {
			t.Fatalf("local websocket backend failed: %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("local websocket backend did not send its response")
	}
}
