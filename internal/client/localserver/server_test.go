package localserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func textHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
}

type closableHandler struct {
	http.Handler
	closed bool
}

func (c *closableHandler) Close() error {
	c.closed = true
	return nil
}

func get(t *testing.T, server *Server, host string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Host = host
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestServerRoutesBySubdomain(t *testing.T) {
	server := New("test server")
	if err := server.Register("a", textHandler("A")); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := server.Register("b", textHandler("B")); err != nil {
		t.Fatalf("register b: %v", err)
	}

	if body := get(t, server, "a.example.test").Body.String(); body != "A" {
		t.Fatalf("expected A, got %q", body)
	}
	if body := get(t, server, "b.example.test:8001").Body.String(); body != "B" {
		t.Fatalf("expected B for a host with a port, got %q", body)
	}
	if code := get(t, server, "nope.example.test").Code; code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unknown host, got %d", code)
	}
}

func TestServerFallsBackToSoleHandler(t *testing.T) {
	// A direct request to the loopback address carries no subdomain.
	server := New("test server")
	if err := server.Register("only", textHandler("ONLY")); err != nil {
		t.Fatalf("register: %v", err)
	}

	if body := get(t, server, "127.0.0.1:54321").Body.String(); body != "ONLY" {
		t.Fatalf("expected the sole handler to answer, got %q", body)
	}
}

func TestServerAnswersHealthCheckWithoutRoute(t *testing.T) {
	server := New("test server")

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Portr-Ping-Request", "true")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for a ping, got %d", recorder.Code)
	}
}

func TestServerRejectsDuplicateAndEmptySubdomain(t *testing.T) {
	server := New("test server")
	if err := server.Register("dup", textHandler("x")); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := server.Register("dup", textHandler("y"))
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected a duplicate error, got %v", err)
	}
	if err := server.Register("  ", textHandler("z")); err == nil {
		t.Fatalf("expected an error for an empty subdomain")
	}
	if err := server.Register("nil-handler", nil); err == nil {
		t.Fatalf("expected an error for a nil handler")
	}
}

func TestServerClosesHandlersOnUnregisterAndShutdown(t *testing.T) {
	server := New("test server")

	first := &closableHandler{Handler: textHandler("A")}
	second := &closableHandler{Handler: textHandler("B")}
	if err := server.Register("a", first); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := server.Register("b", second); err != nil {
		t.Fatalf("register b: %v", err)
	}

	server.Unregister("a")
	if !first.closed {
		t.Fatal("expected unregister to close the handler")
	}
	if second.closed {
		t.Fatal("unregister must not touch other handlers")
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if !second.closed {
		t.Fatal("expected shutdown to close remaining handlers")
	}
}

func TestServerStartIsIdempotentAndReportsPort(t *testing.T) {
	server := New("test server")
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })

	if server.Port() != 0 || server.Addr() != "" {
		t.Fatal("expected no port before start")
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	port := server.Port()
	if port == 0 {
		t.Fatal("expected an assigned port after start")
	}
	if err := server.Start(); err != nil {
		t.Fatalf("second start: %v", err)
	}
	if server.Port() != port {
		t.Fatal("expected start to be idempotent")
	}
	if !strings.HasPrefix(server.Addr(), "127.0.0.1:") {
		t.Fatalf("expected a loopback address, got %q", server.Addr())
	}
}
