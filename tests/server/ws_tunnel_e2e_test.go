package server_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	clientconfig "github.com/amalshaji/portr/internal/client/config"
	clientdb "github.com/amalshaji/portr/internal/client/db"
	tunnelclient "github.com/amalshaji/portr/internal/client/tunnel"
	"github.com/amalshaji/portr/internal/constants"
	adminmodels "github.com/amalshaji/portr/internal/server/admin/models"
	serverconfig "github.com/amalshaji/portr/internal/server/config"
	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/amalshaji/portr/internal/server/wstunnel"
	netwebsocket "golang.org/x/net/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWebSocketTunnelProxiesHTTPToLocalService(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	user := CreateTestUser(t, db, "tunnel@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "Tunnel Team", user, "admin")

	subdomain := "alpha"
	reserved := adminmodels.NewConnection(adminmodels.ConnectionTypeHTTP, &subdomain, teamUser)
	if err := db.Create(reserved).Error; err != nil {
		t.Fatalf("create reserved connection: %v", err)
	}

	echoWebSocket := netwebsocket.Handler(func(conn *netwebsocket.Conn) {
		var message string
		if err := netwebsocket.Message.Receive(conn, &message); err != nil {
			return
		}
		_ = netwebsocket.Message.Send(conn, "echo:"+message)
	})
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/socket" {
			echoWebSocket.ServeHTTP(w, r)
			return
		}
		w.Header().Set("X-Local-Service", "ok")
		_, _ = w.Write([]byte("local:" + r.URL.Path))
	}))
	defer local.Close()

	localURL, err := url.Parse(local.URL)
	if err != nil {
		t.Fatalf("parse local url: %v", err)
	}
	localHost, localPortRaw, _ := strings.Cut(localURL.Host, ":")
	localPort, err := strconv.Atoi(localPortRaw)
	if err != nil {
		t.Fatalf("parse local port: %v", err)
	}

	manager := wstunnel.New(&serverconfig.Config{}, service.New(&serverdb.Db{Conn: db}))
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_portr/tunnel/connect" {
			manager.Handler().ServeHTTP(w, r)
			return
		}
		conn, initial, err := wstunnel.HijackRequest(w, r)
		if err != nil {
			t.Fatalf("hijack request: %v", err)
		}
		manager.OpenHTTPStream(subdomain, conn, initial)
	}))
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	clientDB := newClientTestDB(t)
	tunnelClient := tunnelclient.New(clientconfig.ClientConfig{
		ServerUrl:             "localhost:1",
		WsUrl:                 proxyURL.Host,
		TunnelUrl:             proxyURL.Host,
		SecretKey:             teamUser.SecretKey,
		ConnectionID:          reserved.ID,
		UseLocalHost:          true,
		EnableRequestLogging:  true,
		HealthCheckInterval:   60,
		HealthCheckMaxRetries: 1,
		Tunnel: clientconfig.Tunnel{
			Name:      "alpha",
			Type:      constants.Http,
			Subdomain: subdomain,
			Host:      localHost,
			Port:      localPort,
			PoolSize:  1,
		},
	}, clientDB, nil, func(err error) {
		t.Errorf("tunnel client failed: %v", err)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = tunnelClient.Start(ctx)
	}()
	defer tunnelClient.Shutdown(context.Background())

	deadline := time.Now().Add(5 * time.Second)
	for !manager.HasHTTPBackend(subdomain) {
		if time.Now().After(deadline) {
			t.Fatal("websocket tunnel did not register")
		}
		time.Sleep(25 * time.Millisecond)
	}

	req, err := http.NewRequest(http.MethodGet, proxy.URL+"/hello", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Host = subdomain + ".localhost"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request through websocket tunnel: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(body))
	}
	if got := string(body); got != "local:/hello" {
		t.Fatalf("expected local service response, got %q", got)
	}
	if resp.Header.Get("X-Local-Service") != "ok" {
		t.Fatalf("expected response header from local service")
	}

	deadline = time.Now().Add(5 * time.Second)
	for {
		var request clientdb.Request
		err := clientDB.Conn.Where("subdomain = ? AND url = ?", subdomain, "/hello").First(&request).Error
		if err == nil {
			if request.Method != http.MethodGet {
				t.Fatalf("expected logged method GET, got %q", request.Method)
			}
			if request.ResponseStatusCode != http.StatusOK {
				t.Fatalf("expected logged status 200, got %d", request.ResponseStatusCode)
			}
			if got := string(request.ResponseBody); got != "local:/hello" {
				t.Fatalf("expected logged response body, got %q", got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected proxied request to be logged: %v", err)
		}
		time.Sleep(25 * time.Millisecond)
	}

	wsConn, err := netwebsocket.Dial("ws://"+proxyURL.Host+"/socket", "", "http://"+proxyURL.Host+"/")
	if err != nil {
		t.Fatalf("dial websocket through tunnel: %v", err)
	}
	if err := netwebsocket.Message.Send(wsConn, "ping"); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}
	var wsMessage string
	if err := netwebsocket.Message.Receive(wsConn, &wsMessage); err != nil {
		t.Fatalf("receive websocket message: %v", err)
	}
	_ = wsConn.Close()
	if wsMessage != "echo:ping" {
		t.Fatalf("expected websocket echo, got %q", wsMessage)
	}

	deadline = time.Now().Add(5 * time.Second)
	for {
		var sessions int64
		if err := clientDB.Conn.Model(&clientdb.WebSocketSession{}).
			Where("subdomain = ? AND url = ?", subdomain, "/socket").
			Count(&sessions).Error; err != nil {
			if strings.Contains(err.Error(), "locked") && time.Now().Before(deadline) {
				time.Sleep(25 * time.Millisecond)
				continue
			}
			t.Fatalf("count websocket sessions: %v", err)
		}
		var events int64
		if err := clientDB.Conn.Model(&clientdb.WebSocketEvent{}).Count(&events).Error; err != nil {
			if strings.Contains(err.Error(), "locked") && time.Now().Before(deadline) {
				time.Sleep(25 * time.Millisecond)
				continue
			}
			t.Fatalf("count websocket events: %v", err)
		}
		if sessions == 1 && events >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected websocket session and events to be logged, sessions=%d events=%d", sessions, events)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestWebSocketTunnelRoundRobinsHTTPPool(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	user := CreateTestUser(t, db, "pool@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "Pool Team", user, "admin")

	subdomain := "pool"
	reserved := adminmodels.NewConnection(adminmodels.ConnectionTypeHTTP, &subdomain, teamUser)
	if err := db.Create(reserved).Error; err != nil {
		t.Fatalf("create reserved connection: %v", err)
	}

	localA, hostA, portA := newHTTPBodyServer(t, "local-a")
	defer localA.Close()
	localB, hostB, portB := newHTTPBodyServer(t, "local-b")
	defer localB.Close()

	manager := wstunnel.New(&serverconfig.Config{}, service.New(&serverdb.Db{Conn: db}))
	proxy := newWSTunnelHTTPProxy(t, manager, subdomain)
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	clientDB := newClientTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startHTTPClient(t, ctx, manager, clientDB, proxyURL.Host, teamUser.SecretKey, reserved.ID, subdomain, hostA, portA)
	startHTTPClient(t, ctx, manager, clientDB, proxyURL.Host, teamUser.SecretKey, reserved.ID, subdomain, hostB, portB)

	deadline := time.Now().Add(5 * time.Second)
	for {
		activeCount := manager.HTTPBackendCount(subdomain)
		if activeCount == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 2 websocket http backends, got %d", activeCount)
		}
		time.Sleep(25 * time.Millisecond)
	}

	got := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		req, err := http.NewRequest(http.MethodGet, proxy.URL+"/pool", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Host = subdomain + ".localhost"
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request through websocket tunnel: %v", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			t.Fatalf("read response: %v", readErr)
		}
		got = append(got, string(body))
	}

	if strings.Join(got, ",") != "local-a,local-b,local-a,local-b" {
		t.Fatalf("expected round-robin responses, got %v", got)
	}
}

func TestWebSocketTunnelProxiesTCPToLocalService(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	user := CreateTestUser(t, db, "tcp@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "TCP Team", user, "admin")

	reserved := adminmodels.NewConnection(adminmodels.ConnectionTypeTCP, nil, teamUser)
	if err := db.Create(reserved).Error; err != nil {
		t.Fatalf("create reserved connection: %v", err)
	}

	localListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen local tcp: %v", err)
	}
	defer localListener.Close()
	go func() {
		for {
			conn, err := localListener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(conn)
		}
	}()
	localHost, localPortRaw, _ := strings.Cut(localListener.Addr().String(), ":")
	localPort, err := strconv.Atoi(localPortRaw)
	if err != nil {
		t.Fatalf("parse local tcp port: %v", err)
	}

	manager := wstunnel.New(&serverconfig.Config{}, service.New(&serverdb.Db{Conn: db}))
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_portr/tunnel/connect" {
			manager.Handler().ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	clientDB := newClientTestDB(t)
	tunnelClient := tunnelclient.New(clientconfig.ClientConfig{
		ServerUrl:             "localhost:1",
		WsUrl:                 proxyURL.Host,
		TunnelUrl:             proxyURL.Host,
		SecretKey:             teamUser.SecretKey,
		ConnectionID:          reserved.ID,
		UseLocalHost:          true,
		HealthCheckInterval:   60,
		HealthCheckMaxRetries: 1,
		Tunnel: clientconfig.Tunnel{
			Name: "tcp",
			Type: constants.Tcp,
			Host: localHost,
			Port: localPort,
		},
	}, clientDB, nil, func(err error) {
		t.Errorf("tunnel client failed: %v", err)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = tunnelClient.Start(ctx)
	}()
	defer tunnelClient.Shutdown(context.Background())

	var remotePort uint32
	deadline := time.Now().Add(5 * time.Second)
	for {
		var stored adminmodels.Connection
		if err := db.First(&stored, "id = ?", reserved.ID).Error; err != nil {
			t.Fatalf("load tcp connection: %v", err)
		}
		if stored.Port != nil && stored.Status == "active" {
			remotePort = *stored.Port
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("tcp websocket tunnel did not register")
		}
		time.Sleep(25 * time.Millisecond)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(remotePort))), 5*time.Second)
	if err != nil {
		t.Fatalf("dial tcp tunnel: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("tcp-ok")); err != nil {
		t.Fatalf("write tcp tunnel: %v", err)
	}
	buf := make([]byte, len("tcp-ok"))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read tcp tunnel: %v", err)
	}
	if string(buf) != "tcp-ok" {
		t.Fatalf("expected tcp echo, got %q", string(buf))
	}
}

func newClientTestDB(t *testing.T) *clientdb.Db {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open client db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get client db connection: %v", err)
	}
	for _, pragma := range clientdb.SQLITE_PRAGMAS {
		if _, err := sqlDB.Exec(pragma); err != nil {
			t.Fatalf("set client db pragma %q: %v", pragma, err)
		}
	}

	if err := db.AutoMigrate(&clientdb.Request{}, &clientdb.WebSocketSession{}, &clientdb.WebSocketEvent{}); err != nil {
		t.Fatalf("migrate client db: %v", err)
	}

	return &clientdb.Db{Conn: db}
}

func newHTTPBodyServer(t *testing.T, body string) (*httptest.Server, string, int) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	parsed, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		t.Fatalf("parse local server url: %v", err)
	}
	host, portRaw, _ := strings.Cut(parsed.Host, ":")
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		server.Close()
		t.Fatalf("parse local server port: %v", err)
	}
	return server, host, port
}

func newWSTunnelHTTPProxy(t *testing.T, manager *wstunnel.Manager, subdomain string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_portr/tunnel/connect" {
			manager.Handler().ServeHTTP(w, r)
			return
		}
		conn, initial, err := wstunnel.HijackRequest(w, r)
		if err != nil {
			t.Fatalf("hijack request: %v", err)
		}
		manager.OpenHTTPStream(subdomain, conn, initial)
	}))
}

func startHTTPClient(
	t *testing.T,
	ctx context.Context,
	manager *wstunnel.Manager,
	clientDB *clientdb.Db,
	proxyHost string,
	secretKey string,
	connectionID string,
	subdomain string,
	localHost string,
	localPort int,
) *tunnelclient.Client {
	t.Helper()

	tunnelClient := tunnelclient.New(clientconfig.ClientConfig{
		ServerUrl:             "localhost:1",
		WsUrl:                 proxyHost,
		TunnelUrl:             proxyHost,
		SecretKey:             secretKey,
		ConnectionID:          connectionID,
		UseLocalHost:          true,
		EnableRequestLogging:  true,
		HealthCheckInterval:   60,
		HealthCheckMaxRetries: 1,
		Tunnel: clientconfig.Tunnel{
			Name:      subdomain,
			Type:      constants.Http,
			Subdomain: subdomain,
			Host:      localHost,
			Port:      localPort,
			PoolSize:  1,
		},
	}, clientDB, nil, func(err error) {
		t.Errorf("tunnel client failed: %v", err)
	})

	go func() {
		_ = tunnelClient.Start(ctx)
	}()
	t.Cleanup(func() {
		tunnelClient.Shutdown(context.Background())
	})

	deadline := time.Now().Add(5 * time.Second)
	for manager.HTTPBackendCount(subdomain) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("websocket tunnel did not register")
		}
		time.Sleep(25 * time.Millisecond)
	}

	return tunnelClient
}
