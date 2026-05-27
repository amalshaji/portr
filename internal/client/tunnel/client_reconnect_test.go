package tunnel

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

func TestReconnectKeepsReadyWebSocketOpen(t *testing.T) {
	closed := make(chan struct{}, 1)

	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		if err := wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeReady}); err != nil {
			return
		}

		var payload string
		_ = websocket.Message.Receive(conn, &payload)
		closed <- struct{}{}
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	client := New(clientcfg.ClientConfig{
		ServerUrl:             host,
		WsUrl:                 host,
		SecretKey:             "sk",
		ConnectionID:          "conn-1",
		UseLocalHost:          true,
		DisableTerminalLogs:   true,
		HealthCheckInterval:   1,
		HealthCheckMaxRetries: 1,
		Tunnel: clientcfg.Tunnel{
			Name:      "demo",
			Type:      constants.Http,
			Host:      "127.0.0.1",
			Port:      3000,
			Subdomain: "demo",
		},
	}, nil, nil, nil)

	if err := client.Reconnect(); err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}

	select {
	case <-closed:
		t.Fatal("expected reconnected websocket to remain open after ready")
	case <-time.After(100 * time.Millisecond):
	}

	if err := client.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket to close after shutdown")
	}
}
