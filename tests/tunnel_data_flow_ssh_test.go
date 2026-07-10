package tests_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	clientconfig "github.com/amalshaji/portr/internal/client/config"
	clientdb "github.com/amalshaji/portr/internal/client/db"
	clientssh "github.com/amalshaji/portr/internal/client/ssh"
	"github.com/amalshaji/portr/internal/constants"
	serverconfig "github.com/amalshaji/portr/internal/server/config"
	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/service"
	sshd "github.com/amalshaji/portr/internal/server/ssh"
	sshserver "github.com/gliderlabs/ssh"
	"gorm.io/gorm"
)

const (
	sshDataFlowConnectionID = "ci-ssh-data-flow-connection"
	sshDataFlowSecretKey    = "ci-ssh-data-flow-secret"
	sshDataFlowSubdomain    = "ci-ssh-data-flow"
	sshDataFlowPublicHost   = sshDataFlowSubdomain + ".example.test"
)

type sshTunnelHarness struct {
	proxy     *proxy.Proxy
	client    *clientssh.SshClient
	cancel    context.CancelFunc
	clientErr chan error
	sshServer *sshserver.Server
	sshErr    chan error
	serverDB  *gorm.DB
}

func startSSHTunnelHarness(t *testing.T, backendURL string) *sshTunnelHarness {
	t.Helper()

	serverDatabase := openTestDatabase(t, "ssh-server", &serverdb.TeamUser{}, &serverdb.Connection{})
	teamUser := serverdb.TeamUser{SecretKey: sshDataFlowSecretKey, Role: "member"}
	if err := serverDatabase.Create(&teamUser).Error; err != nil {
		t.Fatalf("create tunnel user: %v", err)
	}
	subdomain := sshDataFlowSubdomain
	connection := serverdb.Connection{
		ID:          sshDataFlowConnectionID,
		Type:        string(constants.Http),
		Subdomain:   &subdomain,
		Status:      "reserved",
		CreatedByID: teamUser.ID,
	}
	if err := serverDatabase.Create(&connection).Error; err != nil {
		t.Fatalf("create reserved connection: %v", err)
	}

	serverConfig := &serverconfig.Config{
		Ssh:    serverconfig.SshConfig{Host: "127.0.0.1"},
		Proxy:  serverconfig.ProxyConfig{Host: "127.0.0.1"},
		Domain: "example.test",
		Debug:  true,
	}
	proxyServer := proxy.New(serverConfig)
	sshServer := sshd.New(
		&serverConfig.Ssh,
		proxyServer,
		service.New(&serverdb.Db{Conn: serverDatabase}),
	).Build()
	sshListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for SSH server: %v", err)
	}
	sshErr := make(chan error, 1)
	go func() {
		sshErr <- sshServer.Serve(sshListener)
	}()

	clientDatabase := openTestDatabase(
		t,
		"ssh-client",
		&clientdb.Request{},
		&clientdb.WebSocketSession{},
		&clientdb.WebSocketEvent{},
	)
	backendHost, backendPort := backendAddress(t, backendURL)
	client := clientssh.New(clientconfig.ClientConfig{
		SshUrl:                          sshListener.Addr().String(),
		TunnelUrl:                       "example.test",
		SecretKey:                       sshDataFlowSecretKey,
		ConnectionID:                    sshDataFlowConnectionID,
		HealthCheckInterval:             60,
		HealthCheckMaxRetries:           1,
		DisableTerminalLogs:             true,
		InsecureSkipHostKeyVerification: true,
		EnableRequestLogging:            true,
		RedactHeaders:                   append([]string(nil), clientconfig.DefaultRedactHeaders...),
		Tunnel: clientconfig.Tunnel{
			Name:      "ci-ssh-data-flow",
			Subdomain: sshDataFlowSubdomain,
			Host:      backendHost,
			Port:      backendPort,
			Type:      constants.Http,
		},
	}, &clientdb.Db{Conn: clientDatabase}, nil, nil)

	started := make(chan struct{}, 1)
	client.SetEventHandler(func(event clientssh.Event) {
		if event.Type == clientssh.EventStarted {
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
		_ = sshServer.Close()
		t.Fatalf("SSH tunnel client stopped before becoming ready: %v", err)
	case <-time.After(testTimeout):
		cancel()
		_ = sshServer.Close()
		t.Fatal("timed out waiting for SSH tunnel client readiness")
	}

	harness := &sshTunnelHarness{
		proxy:     proxyServer,
		client:    client,
		cancel:    cancel,
		clientErr: clientErr,
		sshServer: sshServer,
		sshErr:    sshErr,
		serverDB:  serverDatabase,
	}
	t.Cleanup(func() { harness.close(t) })
	return harness
}

func (h *sshTunnelHarness) close(t *testing.T) {
	t.Helper()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), testTimeout)
	defer cancelShutdown()
	if err := h.client.Shutdown(shutdownContext); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!strings.Contains(strings.ToLower(err.Error()), "closed network connection") {
		t.Errorf("shut down SSH tunnel client: %v", err)
	}
	h.cancel()
	select {
	case err := <-h.clientErr:
		if err != nil {
			t.Errorf("SSH tunnel client exited with error: %v", err)
		}
	case <-time.After(testTimeout):
		t.Error("timed out waiting for SSH tunnel client shutdown")
	}
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

func (h *sshTunnelHarness) waitForConnectionClosed(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		var connection serverdb.Connection
		if err := h.serverDB.First(&connection, "id = ?", sshDataFlowConnectionID).Error; err == nil && connection.Status == "closed" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("timed out waiting for SSH connection to be marked closed")
}

func TestSSHTunnelDataFlowHTTP(t *testing.T) {
	releaseResponse := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseResponse) }) }
	t.Cleanup(release)

	requestSeen := make(chan error, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err == nil && string(body) != "request-through-ssh-tunnel" {
			err = fmt.Errorf("unexpected request body %q", body)
		}
		if err == nil && r.Header.Get("X-Data-Flow") != "ssh-ci" {
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

	harness := startSSHTunnelHarness(t, backend.URL)
	publicServer := httptest.NewServer(harness.proxy)
	defer publicServer.Close()

	request, err := http.NewRequest(http.MethodPost, publicServer.URL+"/stream?source=ssh-ci", strings.NewReader("request-through-ssh-tunnel"))
	if err != nil {
		t.Fatalf("create public request: %v", err)
	}
	request.Host = sshDataFlowPublicHost
	request.Header.Set("X-Data-Flow", "ssh-ci")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send public request through SSH tunnel: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}
	buffer := make([]byte, len("first-chunk"))
	if _, err := io.ReadFull(response.Body, buffer); err != nil {
		t.Fatalf("read streamed first chunk: %v", err)
	}
	if string(buffer) != "first-chunk" {
		t.Fatalf("unexpected first chunk %q", string(buffer))
	}
	release()
	bodyRemainder, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(bodyRemainder) != "second-chunk" {
		t.Fatalf("unexpected response body remainder %q", string(bodyRemainder))
	}
	if response.Trailer.Get("X-Tunnel-Trailer") != "complete" {
		t.Fatalf("expected trailer to be preserved, got %q", response.Trailer.Get("X-Tunnel-Trailer"))
	}

	select {
	case err := <-requestSeen:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(testTimeout):
		t.Fatal("backend did not receive request")
	}
}
