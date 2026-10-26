package tunnel

import (
	"context"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/hashicorp/yamux"
	"golang.org/x/net/websocket"
)

func TestHealthCheckTimesOutWhenServerStopsResponding(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		// Completes the handshake but never runs yamux, so pings go unanswered.
		_ = wsproto.NewWriter(c).Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.ProtocolVersion})
		<-release
	}))
	defer server.Close()
	defer close(release)
	host := strings.TrimPrefix(server.URL, "http://")
	s, err := Connect(context.Background(), clientcfg.ClientConfig{ServerUrl: host, WsUrl: host, UseLocalHost: true}, "conn")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	checked := make(chan error, 1)
	go func() { checked <- s.HealthCheck(30 * time.Millisecond) }()
	select {
	case err := <-checked:
		if err == nil {
			t.Error("expected health check failure")
		}
	case <-time.After(time.Second):
		t.Error("HealthCheck exceeded its timeout")
	}
	if _, err := s.Accept(); err == nil {
		t.Error("expected a timed-out session to be closed")
	}
}

// slowLinkConn makes every frame take a few milliseconds to send, so a pong
// queues behind data frames from many busy streams while bytes keep arriving.
type slowLinkConn struct {
	net.Conn
}

func (c slowLinkConn) Write(p []byte) (int, error) {
	time.Sleep(5 * time.Millisecond)
	return c.Conn.Write(p)
}

func TestHealthCheckAcceptsTrafficWhenPongIsQueued(t *testing.T) {
	for _, tc := range []struct {
		name    string
		streams int
		healthy bool
	}{
		{name: "busy tunnel", streams: 32, healthy: true},
		{name: "silent tunnel", streams: 0, healthy: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			release := make(chan struct{})
			start := make(chan struct{})
			server := httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
				_ = wsproto.NewWriter(c).Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.ProtocolVersion})
				if tc.streams == 0 {
					// Never answers pings.
					<-release
					return
				}
				mux, err := yamux.Server(slowLinkConn{wsproto.MuxConn(c)}, wsproto.MuxConfig(discardLogger))
				if err != nil {
					return
				}
				defer mux.Close()
				streams := make([]net.Conn, 0, tc.streams)
				for range tc.streams {
					stream, err := mux.OpenStream()
					if err != nil {
						return
					}
					streams = append(streams, stream)
				}
				// Start writing only once every stream is open, so stream
				// setup does not queue behind data.
				<-start
				for _, stream := range streams {
					go func() {
						for {
							if _, err := stream.Write(make([]byte, 1024)); err != nil {
								return
							}
						}
					}()
				}
				<-mux.CloseChan()
			}))
			defer server.Close()
			defer close(release)
			host := strings.TrimPrefix(server.URL, "http://")
			s, err := Connect(context.Background(), clientcfg.ClientConfig{ServerUrl: host, WsUrl: host, UseLocalHost: true}, "conn")
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			for range tc.streams {
				stream, err := s.Accept()
				if err != nil {
					t.Fatal(err)
				}
				go func() { _, _ = io.Copy(io.Discard, stream) }()
			}
			close(start)
			time.Sleep(50 * time.Millisecond)

			err = s.HealthCheck(100 * time.Millisecond)
			if tc.healthy && err != nil {
				t.Fatalf("expected a busy tunnel to stay healthy, got %v", err)
			}
			if !tc.healthy && err == nil {
				t.Fatal("expected a tunnel with no traffic and no pong to fail")
			}
		})
	}
}
