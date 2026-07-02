package ssh

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	clientdb "github.com/amalshaji/portr/internal/client/db"
)

func startClosingTCPServer(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	return listener.Addr().String()
}

func runTunnelRequest(t *testing.T, localEndpoint, rawRequest string) string {
	t.Helper()
	remoteConn, clientConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close() })
	client := &SshClient{config: clientcfg.ClientConfig{Tunnel: clientcfg.Tunnel{
		Name: "test", Subdomain: "test", Port: 3000,
	}}}

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

func assertLocalServerUnavailable(t *testing.T, response string) {
	t.Helper()
	if !strings.HasPrefix(response, "HTTP/1.1 503 Service Unavailable") ||
		!strings.Contains(response, "X-Portr-Error: true") ||
		!strings.Contains(response, "X-Portr-Error-Reason: local-server-not-online") {
		t.Fatalf("unexpected response %q", response)
	}
}

func TestHTTPTunnelReturns503WhenLocalClosesWithoutResponse(t *testing.T) {
	response := runTunnelRequest(t, startClosingTCPServer(t), "GET / HTTP/1.1\r\nHost: test.example\r\nConnection: close\r\n\r\n")
	assertLocalServerUnavailable(t, response)
}

func TestWebSocketTunnelReturns503WhenLocalClosesWithoutHandshake(t *testing.T) {
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
	response := runTunnelRequest(t, startClosingTCPServer(t), request)
	assertLocalServerUnavailable(t, response)
}

func TestHTTPTunnelReusesForwardedConnection(t *testing.T) {
	var requests atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, "ok")
	}))
	defer backend.Close()

	remoteConn, clientConn := net.Pipe()
	client := &SshClient{
		config: clientcfg.ClientConfig{Tunnel: clientcfg.Tunnel{Name: "test", Subdomain: "test", Port: 3000}},
		db:     newTestRequestStore(t),
	}
	done := make(chan struct{})
	go func() {
		client.httpTunnel(remoteConn, strings.TrimPrefix(backend.URL, "http://"))
		close(done)
	}()

	reader := bufio.NewReader(clientConn)
	for index := 0; index < 2; index++ {
		connection := "keep-alive"
		if index == 1 {
			connection = "close"
		}
		request, _ := http.NewRequest(http.MethodGet, "http://test.example/", nil)
		request.Header.Set("Connection", connection)
		if err := request.Write(clientConn); err != nil {
			t.Fatalf("write request %d: %v", index, err)
		}
		response, err := http.ReadResponse(reader, request)
		if err != nil {
			t.Fatalf("read response %d: %v", index, err)
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil || string(body) != "ok" {
			t.Fatalf("response %d body=%q err=%v", index, body, err)
		}
	}
	_ = clientConn.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP tunnel did not close")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown tunnel client: %v", err)
	}
	if requests.Load() != 2 {
		t.Fatalf("expected two requests over one forward, got %d", requests.Load())
	}
	var persisted int64
	if err := client.db.Conn.Model(&clientdb.Request{}).Count(&persisted).Error; err != nil {
		t.Fatalf("count persisted requests: %v", err)
	}
	if persisted != 2 {
		t.Fatalf("expected two persisted requests, got %d", persisted)
	}
}
