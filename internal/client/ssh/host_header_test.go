package ssh

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientdb "github.com/amalshaji/portr/internal/client/db"
	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
)

// hostSeenByBackend runs one request through the HTTP tunnel and returns the
// Host header the local server received.
func hostSeenByBackend(t *testing.T, hostHeader string) string {
	t.Helper()

	received := make(chan string, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		received <- r.Host
	}))
	defer backend.Close()

	client := &SshClient{config: clientcfg.ClientConfig{Tunnel: clientcfg.Tunnel{
		Name: "test", Subdomain: "test", Port: 3000, HostHeader: hostHeader,
	}}}
	runRawRequest(t, client, strings.TrimPrefix(backend.URL, "http://"),
		"GET / HTTP/1.1\r\nHost: test.example\r\nConnection: close\r\n\r\n")

	select {
	case host := <-received:
		return host
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not receive a request")
		return ""
	}
}

func runRawRequest(t *testing.T, client *SshClient, localEndpoint, rawRequest string) string {
	t.Helper()

	remoteConn, clientConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close() })

	done := make(chan struct{})
	go func() {
		client.httpTunnel(remoteConn, localEndpoint)
		close(done)
	}()
	if _, err := clientConn.Write([]byte(rawRequest)); err != nil {
		t.Fatalf("write request: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	response, err := io.ReadAll(clientConn)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "closed") {
		t.Fatalf("read response: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tunnel handler did not finish")
	}
	return string(response)
}

func TestHTTPTunnelPassesPublicHostByDefault(t *testing.T) {
	if host := hostSeenByBackend(t, ""); host != "test.example" {
		t.Fatalf("expected the public host to pass through, got %q", host)
	}
}

func TestHTTPTunnelRewritesHostToLocalAddress(t *testing.T) {
	received := make(chan string, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		received <- r.Host
	}))
	defer backend.Close()
	localEndpoint := strings.TrimPrefix(backend.URL, "http://")

	client := &SshClient{config: clientcfg.ClientConfig{Tunnel: clientcfg.Tunnel{
		Name: "test", Subdomain: "test", Port: 3000, HostHeader: clientcfg.HostHeaderRewrite,
	}}}
	runRawRequest(t, client, localEndpoint,
		"GET / HTTP/1.1\r\nHost: test.example\r\nConnection: close\r\n\r\n")

	select {
	case host := <-received:
		if host != localEndpoint {
			t.Fatalf("expected local endpoint %q, got %q", localEndpoint, host)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not receive a request")
	}
}

func TestHTTPTunnelSendsLiteralHostHeader(t *testing.T) {
	if host := hostSeenByBackend(t, "myapp.local"); host != "myapp.local" {
		t.Fatalf("expected literal host header, got %q", host)
	}
}

func TestHTTPTunnelLogsPublicHostWhenRewriting(t *testing.T) {
	received := make(chan string, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Host
		_, _ = io.WriteString(w, "ok")
	}))
	defer backend.Close()
	localEndpoint := strings.TrimPrefix(backend.URL, "http://")

	client := &SshClient{
		config: clientcfg.ClientConfig{
			EnableRequestLogging: true,
			Tunnel: clientcfg.Tunnel{
				Name: "test", Subdomain: "test", Port: 3000, HostHeader: clientcfg.HostHeaderRewrite,
			},
		},
		db: newTestRequestStore(t),
	}
	runRawRequest(t, client, localEndpoint,
		"GET / HTTP/1.1\r\nHost: test.example\r\nConnection: close\r\n\r\n")

	select {
	case host := <-received:
		if host != localEndpoint {
			t.Fatalf("expected the backend to see %q, got %q", localEndpoint, host)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not receive a request")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown tunnel client: %v", err)
	}

	var logged clientdb.Request
	if err := client.db.Conn.First(&logged).Error; err != nil {
		t.Fatalf("load persisted request: %v", err)
	}
	if logged.Host != "test.example" {
		t.Fatalf("inspector must log the public host, got %q", logged.Host)
	}
}

// startWebSocketEchoServer accepts one connection, records the handshake Host,
// and replies with a 101 so the tunnel treats it as an upgraded session.
func startWebSocketEchoServer(t *testing.T, received chan<- string) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		request, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			return
		}
		received <- request.Host
		_, _ = io.WriteString(conn, strings.Join([]string{
			"HTTP/1.1 101 Switching Protocols",
			"Upgrade: websocket",
			"Connection: Upgrade",
			"Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=",
			"",
			"",
		}, "\r\n"))
		// Hold the connection open briefly so the tunnel can relay the 101.
		time.Sleep(100 * time.Millisecond)
	}()

	return listener.Addr().String()
}

func TestWebSocketTunnelRewritesHostHeader(t *testing.T) {
	received := make(chan string, 1)
	localEndpoint := startWebSocketEchoServer(t, received)

	client := &SshClient{
		config: clientcfg.ClientConfig{
			EnableRequestLogging: true,
			Tunnel: clientcfg.Tunnel{
				Name: "test", Subdomain: "test", Port: 3000, HostHeader: clientcfg.HostHeaderRewrite,
			},
		},
		db: newTestRequestStore(t),
	}

	request := strings.Join([]string{
		"GET /ws HTTP/1.1",
		"Host: test.example",
		"Connection: Upgrade",
		"Upgrade: websocket",
		"Sec-WebSocket-Version: 13",
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
		"",
		"",
	}, "\r\n")
	runRawRequest(t, client, localEndpoint, request)

	select {
	case host := <-received:
		if host != localEndpoint {
			t.Fatalf("expected the local server to see %q, got %q", localEndpoint, host)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("local server did not receive a handshake")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown tunnel client: %v", err)
	}

	// The handshake request object is shared with the capture goroutine, so an
	// in-place rewrite here would corrupt the logged host.
	var session clientdb.WebSocketSession
	if err := client.db.Conn.First(&session).Error; err != nil {
		t.Fatalf("load persisted websocket session: %v", err)
	}
	if session.Host != "test.example" {
		t.Fatalf("inspector must log the public host, got %q", session.Host)
	}
}
