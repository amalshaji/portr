package proxy

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverConfig "github.com/amalshaji/portr/internal/server/config"
)

func TestProxy_AddBackendAndRoundRobin(t *testing.T) {
	cfg := &serverConfig.Config{}
	p := New(cfg)

	sub := "sub"
	if err := p.AddBackend(sub, "127.0.0.1:1000"); err != nil {
		t.Fatalf("add backend 1: %v", err)
	}
	if err := p.AddBackend(sub, "127.0.0.1:1001"); err != nil {
		t.Fatalf("add backend 2: %v", err)
	}
	if err := p.AddBackend(sub, "127.0.0.1:1002"); err != nil {
		t.Fatalf("add backend 3: %v", err)
	}

	// Round robin over three backends
	got1, _ := p.GetNextBackend(sub)
	got2, _ := p.GetNextBackend(sub)
	got3, _ := p.GetNextBackend(sub)
	got4, _ := p.GetNextBackend(sub)

	if got1 != "127.0.0.1:1000" || got2 != "127.0.0.1:1001" || got3 != "127.0.0.1:1002" || got4 != "127.0.0.1:1000" {
		t.Fatalf("round robin order unexpected: %v, %v, %v, %v", got1, got2, got3, got4)
	}
}

func TestProxy_RemoveBackend(t *testing.T) {
	cfg := &serverConfig.Config{}
	p := New(cfg)
	sub := "sub"
	_ = p.AddBackend(sub, "127.0.0.1:1000")
	_ = p.AddBackend(sub, "127.0.0.1:1001")

	// Remove one backend and ensure the next is consistent
	if err := p.RemoveBackend(sub, "127.0.0.1:1000"); err != nil {
		t.Fatalf("remove backend: %v", err)
	}

	got, err := p.GetNextBackend(sub)
	if err != nil {
		t.Fatalf("get next backend: %v", err)
	}
	if got != "127.0.0.1:1001" {
		t.Fatalf("expected remaining backend, got %v", got)
	}

	// Remove the last backend; follow-up call should error
	if err := p.RemoveBackend(sub, "127.0.0.1:1001"); err != nil {
		t.Fatalf("remove last backend: %v", err)
	}
	if _, err := p.GetNextBackend(sub); err == nil {
		t.Fatalf("expected error after removing all backends")
	}
}

// sendUpgrade sends a minimal WebSocket-style upgrade request to addr for the
// given host and returns once the response status line has been read.
func sendUpgrade(t *testing.T, addr, host string) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	req := "GET / HTTP/1.1\r\nHost: " + host + "\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write upgrade: %v", err)
	}
	// Read whatever the proxy sends back (101, error page, or EOF) and discard.
	_, _ = bufio.NewReader(conn).ReadString('\n')
}

func newTestProxy(sub, backend string) *Proxy {
	cfg := &serverConfig.Config{Domain: "example.com"}
	p := New(cfg)
	_ = p.AddBackend(sub, backend)
	return p
}

// A WebSocket that upgrades and then closes (EOF) is a normal termination, not a
// backend failure — the backend must stay in the pool.
func TestReverseProxy_KeepsBackendOnNormalClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen backend: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Upgrade, then immediately close so the proxied stream ends in EOF.
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
		conn.Close()
	}()

	p := newTestProxy("sub", ln.Addr().String())
	srv := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer srv.Close()

	sendUpgrade(t, strings.TrimPrefix(srv.URL, "http://"), "sub.example.com")

	if _, err := p.GetNextBackend("sub"); err != nil {
		t.Fatalf("backend evicted on normal close: %v", err)
	}
}

// An unreachable backend never responds — it should be evicted from the pool.
func TestReverseProxy_EvictsUnreachableBackend(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	dead := ln.Addr().String()
	ln.Close() // free the port so dials are refused

	p := newTestProxy("sub", dead)
	srv := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer srv.Close()

	sendUpgrade(t, strings.TrimPrefix(srv.URL, "http://"), "sub.example.com")

	if _, err := p.GetNextBackend("sub"); err == nil {
		t.Fatalf("expected unreachable backend to be evicted")
	}
}
