package tunneltransport

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
)

func TestHTTPTunnelReusesLocalConnectionsAcrossStreams(t *testing.T) {
	var opened atomic.Int32
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}
	}
	backend.Start()
	defer backend.Close()

	client := &Client{config: clientcfg.ClientConfig{Tunnel: clientcfg.Tunnel{
		Name: "test", Subdomain: "test", Port: 3000,
	}}}
	defer client.closeLocalTransport()
	localEndpoint := strings.TrimPrefix(backend.URL, "http://")
	for range 3 {
		response := runRawRequest(t, client, localEndpoint,
			"GET / HTTP/1.1\r\nHost: test.example\r\nConnection: close\r\n\r\n")
		if !strings.HasPrefix(response, "HTTP/1.1 200 OK") {
			t.Fatalf("unexpected response %q", response)
		}
	}
	if got := opened.Load(); got != 1 {
		t.Fatalf("expected one local connection for three streams, got %d", got)
	}
}
