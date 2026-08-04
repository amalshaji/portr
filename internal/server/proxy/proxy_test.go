package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalshaji/portr/internal/server/config"
)

type fakeTunnelBackend struct {
	streamed bool
}

func (*fakeTunnelBackend) Endpoint() string { return "/tunnel/connect" }

func (*fakeTunnelBackend) Handler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
}

func (*fakeTunnelBackend) HasHTTPBackend(subdomain string) bool {
	return subdomain == "demo"
}

func (b *fakeTunnelBackend) ServeHTTPStream(_ string, writer http.ResponseWriter, _ *http.Request) {
	b.streamed = true
	writer.WriteHeader(http.StatusAccepted)
}

func TestProxyDelegatesTunnelEndpointAndHTTPStreams(t *testing.T) {
	backend := &fakeTunnelBackend{}
	proxyServer := New(&config.Config{Domain: "example.com"})
	proxyServer.SetTunnelBackend(backend)

	endpointResponse := httptest.NewRecorder()
	proxyServer.ServeHTTP(endpointResponse, httptest.NewRequest(http.MethodGet, "http://example.com/tunnel/connect", nil))
	if endpointResponse.Code != http.StatusNoContent {
		t.Fatalf("expected tunnel endpoint status %d, got %d", http.StatusNoContent, endpointResponse.Code)
	}

	streamResponse := httptest.NewRecorder()
	proxyServer.ServeHTTP(streamResponse, httptest.NewRequest(http.MethodGet, "http://demo.example.com/hello", nil))
	if streamResponse.Code != http.StatusAccepted {
		t.Fatalf("expected tunnel stream status %d, got %d", http.StatusAccepted, streamResponse.Code)
	}
	if !backend.streamed {
		t.Fatal("expected tunnel backend to serve the request")
	}
}
