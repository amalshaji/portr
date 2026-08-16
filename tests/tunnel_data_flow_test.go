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

	clientdb "github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/tunneltransport"
	clientconfig "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
	serverconfig "github.com/amalshaji/portr/internal/server/config"
	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/service"
	sshd "github.com/amalshaji/portr/internal/server/ssh"
	"github.com/amalshaji/portr/internal/server/wstunnel"
	"github.com/glebarez/sqlite"
	sshserver "github.com/gliderlabs/ssh"
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

// dataFlowTransports is the transport axis of the integration matrix. Every
// data-flow test below runs once per entry, so a new tunnel channel only needs
// one flow function to be covered on both transports.
var dataFlowTransports = []clientconfig.Transport{
	clientconfig.TransportWebSocket,
	clientconfig.TransportSSH,
}

type tunnelHarness struct {
	transport    clientconfig.Transport
	client       *tunneltransport.Client
	cancel       context.CancelFunc
	clientErr    chan error
	publicServer *httptest.Server
	tunnelHost   string
	serverDB     *gorm.DB

	sshServer *sshserver.Server
	sshErr    chan error
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

func startDataFlowServerDatabase(t *testing.T, tunnelType constants.ConnectionType) *gorm.DB {
	t.Helper()
	serverDatabase := openTestDatabase(t, "server", &serverdb.TeamUser{}, &serverdb.Connection{})
	teamUser := serverdb.TeamUser{SecretKey: testSecretKey, Role: "member"}
	if err := serverDatabase.Create(&teamUser).Error; err != nil {
		t.Fatalf("create tunnel user: %v", err)
	}
	connection := serverdb.Connection{
		ID:          testConnectionID,
		Type:        string(tunnelType),
		Status:      "reserved",
		CreatedByID: teamUser.ID,
	}
	if tunnelType == constants.Http {
		subdomain := testSubdomain
		connection.Subdomain = &subdomain
	}
	if err := serverDatabase.Create(&connection).Error; err != nil {
		t.Fatalf("create reserved connection: %v", err)
	}
	return serverDatabase
}

func startDataFlowClient(t *testing.T, client *tunneltransport.Client, onFail func()) (context.CancelFunc, chan error) {
	t.Helper()
	started := make(chan struct{}, 1)
	client.SetEventHandler(func(event tunneltransport.Event) {
		if event.Type == tunneltransport.EventStarted {
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
		onFail()
		t.Fatalf("tunnel client stopped before becoming ready: %v", err)
	case <-time.After(testTimeout):
		cancel()
		onFail()
		t.Fatal("timed out waiting for tunnel client readiness")
	}
	return cancel, clientErr
}

// startTunnelHarness runs the full stack for one transport: server database,
// public proxy front, transport listener (WebSocket endpoint or SSH server),
// and a real tunneltransport client connected through it.
func startTunnelHarness(t *testing.T, transport clientconfig.Transport, tunnelType constants.ConnectionType, backendHost string, backendPort int) *tunnelHarness {
	t.Helper()

	serverDatabase := startDataFlowServerDatabase(t, tunnelType)
	serverService := service.New(&serverdb.Db{Conn: serverDatabase})
	clientDatabase := openTestDatabase(
		t,
		"client",
		&clientdb.Request{},
		&clientdb.WebSocketSession{},
		&clientdb.WebSocketEvent{},
	)

	clientCfg := clientconfig.ClientConfig{
		Transport:             transport,
		SecretKey:             testSecretKey,
		ConnectionID:          testConnectionID,
		Debug:                 true,
		HealthCheckInterval:   60,
		HealthCheckMaxRetries: 1,
		DisableTerminalLogs:   true,
		EnableRequestLogging:  true,
		RedactHeaders:         append([]string(nil), clientconfig.DefaultRedactHeaders...),
		Tunnel: clientconfig.Tunnel{
			Name:      "ci-data-flow",
			Subdomain: testSubdomain,
			Host:      backendHost,
			Port:      backendPort,
			Type:      tunnelType,
		},
	}

	harness := &tunnelHarness{transport: transport, serverDB: serverDatabase}

	switch transport {
	case clientconfig.TransportWebSocket:
		serverConfig := &serverconfig.Config{
			Proxy:        serverconfig.ProxyConfig{Host: "localhost"},
			Domain:       "localhost",
			UseLocalHost: true,
			Debug:        true,
		}
		proxyServer := proxy.New(serverConfig)
		proxyServer.SetTunnelBackend(wstunnel.New(serverConfig, serverService))
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

		clientCfg.ServerUrl = publicHost
		clientCfg.WsUrl = publicHost
		clientCfg.TunnelUrl = publicHost
		clientCfg.UseLocalHost = true

		harness.publicServer = publicServer
		harness.tunnelHost = testSubdomain + "." + publicHost
		harness.client = tunneltransport.New(clientCfg, &clientdb.Db{Conn: clientDatabase}, nil, nil)
		harness.cancel, harness.clientErr = startDataFlowClient(t, harness.client, publicServer.Close)

	case clientconfig.TransportSSH:
		serverConfig := &serverconfig.Config{
			Ssh:    serverconfig.SshConfig{Host: "127.0.0.1"},
			Proxy:  serverconfig.ProxyConfig{Host: "127.0.0.1"},
			Domain: "example.test",
			Debug:  true,
		}
		proxyServer := proxy.New(serverConfig)
		sshServer := sshd.New(&serverConfig.Ssh, proxyServer, serverService).Build()
		sshListener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen for SSH server: %v", err)
		}
		sshErr := make(chan error, 1)
		go func() {
			sshErr <- sshServer.Serve(sshListener)
		}()
		publicServer := httptest.NewServer(proxyServer)

		clientCfg.SshUrl = sshListener.Addr().String()
		clientCfg.TunnelUrl = "example.test"
		clientCfg.InsecureSkipHostKeyVerification = true

		harness.publicServer = publicServer
		harness.tunnelHost = testSubdomain + ".example.test"
		harness.sshServer = sshServer
		harness.sshErr = sshErr
		harness.client = tunneltransport.New(clientCfg, &clientdb.Db{Conn: clientDatabase}, nil, nil)
		harness.cancel, harness.clientErr = startDataFlowClient(t, harness.client, func() {
			publicServer.Close()
			_ = sshServer.Close()
		})

	default:
		t.Fatalf("unsupported transport %q", transport)
	}

	t.Cleanup(func() { harness.close(t) })
	return harness
}

func startHTTPTunnelHarness(t *testing.T, transport clientconfig.Transport, backendURL string) *tunnelHarness {
	t.Helper()
	backendHost, backendPort := backendAddress(t, backendURL)
	return startTunnelHarness(t, transport, constants.Http, backendHost, backendPort)
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
	if h.sshServer != nil {
		if err := h.sshServer.Close(); err != nil && !errors.Is(err, sshserver.ErrServerClosed) {
			t.Errorf("close SSH server: %v", err)
		}
		select {
		case err := <-h.sshErr:
			if err != nil && !errors.Is(err, sshserver.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
				t.Errorf("SSH server exited with error: %v", err)
			}
		case <-time.After(testTimeout):
			t.Error("timed out waiting for SSH server shutdown")
		}
		h.waitForConnectionClosed(t)
	}
	h.publicServer.Close()
}

func (h *tunnelHarness) waitForConnectionClosed(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		var connection serverdb.Connection
		if err := h.serverDB.First(&connection, "id = ?", testConnectionID).Error; err == nil && connection.Status == "closed" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("timed out waiting for connection to be marked closed")
}

// waitForTCPPort waits until the server assigned the tunnel its public TCP
// port and returns the address to dial.
func (h *tunnelHarness) waitForTCPPort(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		var connection serverdb.Connection
		err := h.serverDB.First(&connection, "id = ?", testConnectionID).Error
		if err == nil && connection.Status == "active" && connection.Port != nil {
			return net.JoinHostPort("127.0.0.1", strconv.Itoa(int(*connection.Port)))
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for the TCP tunnel port to be assigned")
	return ""
}

func TestTunnelDataFlowHTTP(t *testing.T) {
	for _, transport := range dataFlowTransports {
		t.Run(string(transport), func(t *testing.T) {
			runTunnelDataFlowHTTP(t, transport)
		})
	}
}

func runTunnelDataFlowHTTP(t *testing.T, transport clientconfig.Transport) {
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

	harness := startHTTPTunnelHarness(t, transport, backend.URL)
	request, err := http.NewRequest(http.MethodPost, harness.publicServer.URL+"/stream?source=ci", strings.NewReader("request-through-tunnel"))
	if err != nil {
		t.Fatalf("create public request: %v", err)
	}
	request.Host = harness.tunnelHost
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

func TestTunnelDataFlowStreamsRequestBody(t *testing.T) {
	for _, transport := range dataFlowTransports {
		t.Run(string(transport), func(t *testing.T) {
			runTunnelDataFlowStreamsRequestBody(t, transport)
		})
	}
}

func runTunnelDataFlowStreamsRequestBody(t *testing.T, transport clientconfig.Transport) {
	firstChunkSeen := make(chan error, 1)
	requestComplete := make(chan error, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		first := make([]byte, len("first-chunk"))
		_, err := io.ReadFull(r.Body, first)
		if err == nil && string(first) != "first-chunk" {
			err = fmt.Errorf("unexpected first request chunk %q", first)
		}
		firstChunkSeen <- err
		if err != nil {
			return
		}

		rest, err := io.ReadAll(r.Body)
		if err == nil && string(rest) != "second-chunk" {
			err = fmt.Errorf("unexpected final request chunk %q", rest)
		}
		requestComplete <- err
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	harness := startHTTPTunnelHarness(t, transport, backend.URL)
	bodyReader, bodyWriter := io.Pipe()
	t.Cleanup(func() { _ = bodyWriter.Close() })
	request, err := http.NewRequest(http.MethodPost, harness.publicServer.URL+"/upload", bodyReader)
	if err != nil {
		t.Fatalf("create streaming request: %v", err)
	}
	request.Host = harness.tunnelHost

	type responseResult struct {
		response *http.Response
		err      error
	}
	responseCh := make(chan responseResult, 1)
	go func() {
		response, requestErr := http.DefaultClient.Do(request)
		responseCh <- responseResult{response: response, err: requestErr}
	}()

	if _, err := io.WriteString(bodyWriter, "first-chunk"); err != nil {
		t.Fatalf("write first request chunk: %v", err)
	}
	select {
	case err := <-firstChunkSeen:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(testTimeout):
		_ = bodyWriter.Close()
		t.Fatal("request body was buffered instead of streamed through the tunnel")
	}

	if _, err := io.WriteString(bodyWriter, "second-chunk"); err != nil {
		t.Fatalf("write final request chunk: %v", err)
	}
	if err := bodyWriter.Close(); err != nil {
		t.Fatalf("close streaming request body: %v", err)
	}

	select {
	case err := <-requestComplete:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(testTimeout):
		t.Fatal("backend did not receive the complete streaming request")
	}

	select {
	case result := <-responseCh:
		if result.err != nil {
			t.Fatalf("send streaming request through tunnel: %v", result.err)
		}
		defer result.response.Body.Close()
		if result.response.StatusCode != http.StatusNoContent {
			t.Fatalf("unexpected response status %d", result.response.StatusCode)
		}
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for streaming request response")
	}
}

func TestTunnelDataFlowWebSocket(t *testing.T) {
	for _, transport := range dataFlowTransports {
		t.Run(string(transport), func(t *testing.T) {
			runTunnelDataFlowWebSocket(t, transport)
		})
	}
}

func runTunnelDataFlowWebSocket(t *testing.T, transport clientconfig.Transport) {
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

	harness := startHTTPTunnelHarness(t, transport, backend.URL)
	publicURL, err := url.Parse(harness.publicServer.URL)
	if err != nil {
		t.Fatalf("parse public proxy URL: %v", err)
	}
	rawConnection, err := net.DialTimeout("tcp", publicURL.Host, testTimeout)
	if err != nil {
		t.Fatalf("dial public proxy: %v", err)
	}
	websocketConfig, err := websocket.NewConfig("ws://"+harness.tunnelHost+"/echo", "http://"+harness.tunnelHost)
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

func TestTunnelDataFlowTCP(t *testing.T) {
	for _, transport := range dataFlowTransports {
		t.Run(string(transport), func(t *testing.T) {
			runTunnelDataFlowTCP(t, transport)
		})
	}
}

func runTunnelDataFlowTCP(t *testing.T, transport clientconfig.Transport) {
	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen local tcp backend: %v", err)
	}
	defer backendListener.Close()
	go func() {
		for {
			conn, err := backendListener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				request, readErr := io.ReadAll(conn)
				if readErr != nil || string(request) != "tcp-request" {
					return
				}
				_, _ = conn.Write([]byte("tcp-response"))
				if tcpConn, ok := conn.(*net.TCPConn); ok {
					_ = tcpConn.CloseWrite()
				}
			}(conn)
		}
	}()
	backendHost, backendPortText, err := net.SplitHostPort(backendListener.Addr().String())
	if err != nil {
		t.Fatalf("split backend address: %v", err)
	}
	backendPort, err := strconv.Atoi(backendPortText)
	if err != nil {
		t.Fatalf("parse backend port: %v", err)
	}

	harness := startTunnelHarness(t, transport, constants.Tcp, backendHost, backendPort)
	publicAddress := harness.waitForTCPPort(t)

	conn, err := net.DialTimeout("tcp", publicAddress, testTimeout)
	if err != nil {
		t.Fatalf("dial public tcp tunnel: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("tcp-request")); err != nil {
		t.Fatalf("write through tcp tunnel: %v", err)
	}
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		t.Fatalf("expected TCP connection, got %T", conn)
	}
	if err := tcpConn.CloseWrite(); err != nil {
		t.Fatalf("half-close tcp request: %v", err)
	}
	response, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read tcp tunnel response: %v", err)
	}
	if string(response) != "tcp-response" {
		t.Fatalf("unexpected tcp response %q", response)
	}
}
