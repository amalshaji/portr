package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	got1, _ := p.nextBackends(sub, 1)
	got2, _ := p.nextBackends(sub, 1)
	got3, _ := p.nextBackends(sub, 1)
	got4, _ := p.nextBackends(sub, 1)

	if got1[0] != "127.0.0.1:1000" || got2[0] != "127.0.0.1:1001" || got3[0] != "127.0.0.1:1002" || got4[0] != "127.0.0.1:1000" {
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

	got, err := p.nextBackends(sub, 1)
	if err != nil {
		t.Fatalf("get next backend: %v", err)
	}
	if got[0] != "127.0.0.1:1001" {
		t.Fatalf("expected remaining backend, got %v", got)
	}

	// Remove the last backend; follow-up call should error
	if err := p.RemoveBackend(sub, "127.0.0.1:1001"); err != nil {
		t.Fatalf("remove last backend: %v", err)
	}
	if _, err := p.nextBackends(sub, 1); err == nil {
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

	if _, err := p.nextBackends("sub", 1); err != nil {
		t.Fatalf("backend evicted on normal close: %v", err)
	}
}

// The proxy never mutates the route pool — backend lifecycle is owned by the
// SSH layer. Even an unreachable backend stays put so it can auto-recover once
// its local app comes back (the tunnel itself is still up).
func TestReverseProxy_DoesNotEvictUnreachableBackend(t *testing.T) {
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

	if _, err := p.nextBackends("sub", 1); err != nil {
		t.Fatalf("proxy evicted backend it does not own: %v", err)
	}
}

func deadBackendAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func TestProxyStreamsResponseBeforeBackendCompletes(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "first")
		w.(http.Flusher).Flush()
		<-release
		_, _ = io.WriteString(w, "second")
	}))
	defer backend.Close()

	p := newTestProxy("sub", strings.TrimPrefix(backend.URL, "http://"))
	proxyServer := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer proxyServer.Close()

	request, _ := http.NewRequest(http.MethodGet, proxyServer.URL, nil)
	request.Host = "sub.example.com"
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		close(release)
		t.Fatalf("request proxy: %v", err)
	}
	defer response.Body.Close()

	first := make([]byte, len("first"))
	readDone := make(chan error, 1)
	go func() {
		_, readErr := io.ReadFull(response.Body, first)
		readDone <- readErr
	}()
	select {
	case err := <-readDone:
		if err != nil {
			close(release)
			t.Fatalf("read first chunk: %v", err)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("first response chunk was buffered")
	}
	if string(first) != "first" {
		close(release)
		t.Fatalf("unexpected first chunk %q", first)
	}
	close(release)
	rest, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read remaining response: %v", err)
	}
	if string(rest) != "second" {
		t.Fatalf("unexpected remaining response %q", rest)
	}
}

func TestProxyRetriesReplaySafeRequest(t *testing.T) {
	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, "ok")
	}))
	defer backend.Close()

	p := newTestProxy("sub", deadBackendAddress(t))
	_ = p.AddBackend("sub", strings.TrimPrefix(backend.URL, "http://"))
	proxyServer := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer proxyServer.Close()

	request, _ := http.NewRequest(http.MethodGet, proxyServer.URL, nil)
	request.Host = "sub.example.com"
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request proxy: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || calls.Load() != 1 {
		t.Fatalf("expected retry to live backend, status=%d calls=%d", response.StatusCode, calls.Load())
	}
}

func TestProxyDoesNotRetryPost(t *testing.T) {
	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	p := newTestProxy("sub", deadBackendAddress(t))
	_ = p.AddBackend("sub", strings.TrimPrefix(backend.URL, "http://"))
	proxyServer := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer proxyServer.Close()

	request, _ := http.NewRequest(http.MethodPost, proxyServer.URL, strings.NewReader("side-effect"))
	request.Host = "sub.example.com"
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request proxy: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", response.StatusCode)
	}
	if calls.Load() != 0 {
		t.Fatalf("unsafe request retried %d times", calls.Load())
	}
}

func TestProxyDoesNotRetryGetWithBody(t *testing.T) {
	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	p := newTestProxy("sub", deadBackendAddress(t))
	_ = p.AddBackend("sub", strings.TrimPrefix(backend.URL, "http://"))
	proxyServer := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer proxyServer.Close()

	request, _ := http.NewRequest(http.MethodGet, proxyServer.URL, nil)
	request.Body = io.NopCloser(strings.NewReader("body-with-zero-content-length"))
	request.ContentLength = 0
	request.Host = "sub.example.com"
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request proxy: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", response.StatusCode)
	}
	if calls.Load() != 0 {
		t.Fatalf("GET with body was retried %d times", calls.Load())
	}
}

func TestProxyPreservesInformationalAndFinalStatus(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Link", "</style.css>; rel=preload")
		w.WriteHeader(http.StatusEarlyHints)
		w.Header().Del("Link")
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, "teapot")
	}))
	defer backend.Close()

	p := newTestProxy("sub", strings.TrimPrefix(backend.URL, "http://"))
	proxyServer := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer proxyServer.Close()
	request, _ := http.NewRequest(http.MethodGet, proxyServer.URL, nil)
	request.Host = "sub.example.com"
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request proxy: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusTeapot {
		t.Fatalf("expected final status %d, got %d", http.StatusTeapot, response.StatusCode)
	}
}

func TestProxyPreservesResponseTrailers(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Trailer", "X-Checksum")
		_, _ = io.WriteString(w, "payload")
		w.Header().Set("X-Checksum", "complete")
	}))
	defer backend.Close()

	p := newTestProxy("sub", strings.TrimPrefix(backend.URL, "http://"))
	proxyServer := httptest.NewServer(http.HandlerFunc(p.handleRequest))
	defer proxyServer.Close()
	request, _ := http.NewRequest(http.MethodGet, proxyServer.URL, nil)
	request.Host = "sub.example.com"
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request proxy: %v", err)
	}
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.Trailer.Get("X-Checksum") != "complete" {
		t.Fatalf("expected response trailer, got %q", response.Trailer.Get("X-Checksum"))
	}
}
