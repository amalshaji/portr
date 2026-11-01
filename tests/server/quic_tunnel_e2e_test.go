package server_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/client/tunneltransport"
	clientconfig "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
	adminmodels "github.com/amalshaji/portr/internal/server/admin/models"
	serverconfig "github.com/amalshaji/portr/internal/server/config"
	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/amalshaji/portr/internal/server/wstunnel"
	"github.com/amalshaji/portr/internal/tunnel/quicproto"
	"gorm.io/gorm"
)

type quicE2E struct {
	db        *gorm.DB
	manager   atomic.Pointer[wstunnel.Manager]
	proxy     *httptest.Server
	proxyHost string
	// wsURL is what clients use for websocket; tests that must stay on QUIC
	// point it at a dead port so a fallback fails them.
	wsURL string
	key   string
	pin   string
}

// startQUICE2E runs a tunnel manager with both the websocket endpoint and a
// QUIC listener, behind a visitor proxy that routes every request to subdomain.
func startQUICE2E(t *testing.T, subdomain string) *quicE2E {
	t.Helper()
	db, cleanup := NewTestDB(t)
	t.Cleanup(cleanup)

	key, err := quicproto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	pin, err := quicproto.KeyPin(key)
	if err != nil {
		t.Fatal(err)
	}
	env := &quicE2E{db: db, key: key, pin: pin}
	env.manager.Store(wstunnel.New(&serverconfig.Config{}, service.New(&serverdb.Db{Conn: db})))
	env.proxy = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager := env.manager.Load()
		if r.URL.Path == manager.Endpoint() {
			manager.Handler().ServeHTTP(w, r)
			return
		}
		manager.ServeHTTPStream(subdomain, w, r)
	}))
	t.Cleanup(env.proxy.Close)
	proxyURL, err := url.Parse(env.proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	env.proxyHost = proxyURL.Host
	env.wsURL = proxyURL.Host
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = env.manager.Load().Shutdown(ctx)
	})
	return env
}

func (e *quicE2E) listen(t *testing.T, addr string) string {
	t.Helper()
	quicAddr, err := e.manager.Load().ListenQUIC(addr, e.key)
	if err != nil {
		t.Fatalf("listen quic: %v", err)
	}
	return quicAddr.String()
}

func (e *quicE2E) reserve(t *testing.T, email, subdomain string) (*adminmodels.Connection, string) {
	t.Helper()
	user := CreateTestUser(t, e.db, email, false)
	_, teamUser := CreateTeamAndTeamUser(t, e.db, email+" team", user, "admin")
	reserved := adminmodels.NewConnection(adminmodels.ConnectionTypeHTTP, &subdomain, teamUser)
	if err := e.db.Create(reserved).Error; err != nil {
		t.Fatalf("create reserved connection: %v", err)
	}
	return reserved, teamUser.SecretKey
}

func (e *quicE2E) startClient(t *testing.T, transport clientconfig.Transport, quicURL, secretKey, connectionID, subdomain, localHost string, localPort int) {
	t.Helper()
	tunnelClient := tunneltransport.New(clientconfig.ClientConfig{
		ServerUrl:             "localhost:1",
		WsUrl:                 e.wsURL,
		QuicUrl:               quicURL,
		QuicCertSha256:        e.pin,
		TunnelUrl:             e.proxyHost,
		Transport:             transport,
		SecretKey:             secretKey,
		ConnectionID:          connectionID,
		UseLocalHost:          true,
		EnableRequestLogging:  true,
		HealthCheckInterval:   1,
		HealthCheckMaxRetries: 10,
		Tunnel: clientconfig.Tunnel{
			Name:      subdomain,
			Type:      constants.Http,
			Subdomain: subdomain,
			Host:      localHost,
			Port:      localPort,
			PoolSize:  1,
		},
	}, newClientTestDB(t), nil, func(err error) {
		t.Errorf("tunnel client failed: %v", err)
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = tunnelClient.Start(ctx) }()
	t.Cleanup(func() {
		tunnelClient.Shutdown(context.Background())
		cancel()
	})
}

func (e *quicE2E) waitForBackends(t *testing.T, subdomain string, count int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for e.manager.Load().HTTPBackendCount(subdomain) != count {
		if time.Now().After(deadline) {
			t.Fatalf("expected %d backends for %s, got %d", count, subdomain, e.manager.Load().HTTPBackendCount(subdomain))
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (e *quicE2E) get(t *testing.T, subdomain, path string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.proxy.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = subdomain + ".localhost"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func TestQUICTunnelFallsBackToWebSocketWhenUDPIsBlocked(t *testing.T) {
	env := startQUICE2E(t, "fallback")
	reserved, secretKey := env.reserve(t, "fallback@example.com", "fallback")
	local, host, port := newHTTPBodyServer(t, "via websocket")
	defer local.Close()

	// A bound UDP socket that never answers stands in for a blocked port.
	silent, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()

	env.startClient(t, clientconfig.TransportQUIC, silent.LocalAddr().String(), secretKey, reserved.ID, "fallback", host, port)
	env.waitForBackends(t, "fallback", 1, 10*time.Second)
	if got := env.get(t, "fallback", "/"); got != "via websocket" {
		t.Fatalf("response = %q", got)
	}
}

func TestQUICTunnelSharesPoolWithWebSocketClients(t *testing.T) {
	env := startQUICE2E(t, "mixed")
	quicAddr := env.listen(t, "127.0.0.1:0")
	reserved, secretKey := env.reserve(t, "mixed@example.com", "mixed")
	localA, hostA, portA := newHTTPBodyServer(t, "quic")
	defer localA.Close()
	localB, hostB, portB := newHTTPBodyServer(t, "websocket")
	defer localB.Close()

	env.startClient(t, clientconfig.TransportQUIC, quicAddr, secretKey, reserved.ID, "mixed", hostA, portA)
	env.waitForBackends(t, "mixed", 1, 5*time.Second)
	env.startClient(t, clientconfig.TransportWebSocket, quicAddr, secretKey, reserved.ID, "mixed", hostB, portB)
	env.waitForBackends(t, "mixed", 2, 5*time.Second)

	got := make([]string, 0, 4)
	for range 4 {
		got = append(got, env.get(t, "mixed", "/"))
	}
	if strings.Join(got, ",") != "quic,websocket,quic,websocket" {
		t.Fatalf("expected round-robin across transports, got %v", got)
	}
}

func TestQUICTunnelStalledStreamDoesNotBlockOthers(t *testing.T) {
	env := startQUICE2E(t, "stall")
	quicAddr := env.listen(t, "127.0.0.1:0")
	reserved, secretKey := env.reserve(t, "stall@example.com", "stall")
	env.wsURL = "127.0.0.1:1"

	chunk := strings.Repeat("x", 64<<10)
	local, host, port := newLocalServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk" {
			// Far more than the stream and connection windows.
			for range 1024 {
				if _, err := w.Write([]byte(chunk)); err != nil {
					return
				}
			}
			return
		}
		_, _ = w.Write([]byte("fast"))
	}))
	defer local.Close()

	env.startClient(t, clientconfig.TransportQUIC, quicAddr, secretKey, reserved.ID, "stall", host, port)
	env.waitForBackends(t, "stall", 1, 5*time.Second)

	// Start bulk downloads and never read their bodies. QUIC, unlike yamux,
	// also limits the whole connection, so stalled streams must not use it up.
	for range 4 {
		req, err := http.NewRequest(http.MethodGet, env.proxy.URL+"/bulk", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "stall.localhost"
		stalled, err := (&http.Client{Transport: &http.Transport{}}).Do(req)
		if err != nil {
			t.Fatalf("bulk request: %v", err)
		}
		defer stalled.Body.Close()
	}
	// Give the stalled streams time to fill their windows.
	time.Sleep(200 * time.Millisecond)

	started := time.Now()
	for range 20 {
		if got := env.get(t, "stall", "/fast"); got != "fast" {
			t.Fatalf("fast response = %q", got)
		}
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("fast requests took %s behind a stalled stream", elapsed)
	}
}

func TestQUICTunnelReconnectsAfterServerRestart(t *testing.T) {
	env := startQUICE2E(t, "restart")
	quicAddr := env.listen(t, "127.0.0.1:0")
	reserved, secretKey := env.reserve(t, "restart@example.com", "restart")
	env.wsURL = "127.0.0.1:1"
	local, host, port := newHTTPBodyServer(t, "still here")
	defer local.Close()

	env.startClient(t, clientconfig.TransportQUIC, quicAddr, secretKey, reserved.ID, "restart", host, port)
	env.waitForBackends(t, "restart", 1, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := env.manager.Load().Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	env.manager.Store(wstunnel.New(&serverconfig.Config{}, service.New(&serverdb.Db{Conn: env.db})))
	env.listen(t, quicAddr)

	env.waitForBackends(t, "restart", 1, 15*time.Second)
	if got := env.get(t, "restart", "/"); got != "still here" {
		t.Fatalf("response after restart = %q", got)
	}
}

func newLocalServer(t *testing.T, handler http.Handler) (*httptest.Server, string, int) {
	t.Helper()
	server := httptest.NewServer(handler)
	addr := server.Listener.Addr().(*net.TCPAddr)
	return server, addr.IP.String(), addr.Port
}
