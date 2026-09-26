package tunnel

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

func TestHealthTimeoutIncludesBlockedWriter(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		wsproto.NewWriter(c).Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.ProtocolVersion})
		<-release
	}))
	defer server.Close()
	defer close(release)
	host := strings.TrimPrefix(server.URL, "http://")
	s, err := Connect(context.Background(), clientcfg.ClientConfig{ServerUrl: host, WsUrl: host, UseLocalHost: true}, "conn")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.conn.SetDeadline(time.Now()); s.Close() }()
	started := make(chan struct{})
	go func() {
		close(started)
		for range 512 {
			if s.writer.Send(wsproto.Frame{Type: wsproto.TypeData, Data: make([]byte, 32*1024)}) != nil {
				return
			}
		}
	}()
	<-started
	time.Sleep(300 * time.Millisecond)
	checked := make(chan error, 1)
	go func() { checked <- s.HealthCheck(30 * time.Millisecond) }()
	select {
	case err := <-checked:
		if err == nil {
			t.Error("expected health check failure")
		}
	case <-time.After(time.Second):
		t.Error("HealthCheck exceeded timeout while writer was blocked")
	}
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Error("Close did not interrupt the blocked writer")
	}
}
