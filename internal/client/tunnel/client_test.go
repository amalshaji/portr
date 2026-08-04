package tunnel

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

func TestSessionConnectsAndCarriesStreams(t *testing.T) {
	requestChecked := make(chan struct{}, 1)
	clientFrames := make(chan wsproto.Frame, 4)
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		request := conn.Request()
		if request.URL.Path != wsproto.ConnectPath {
			t.Errorf("unexpected websocket path %q", request.URL.Path)
		}
		if got := request.Header.Get(wsproto.ConnectionIDHeader); got != "conn-1" {
			t.Errorf("unexpected connection ID header %q", got)
		}
		if got := request.Header.Get(wsproto.SecretKeyHeader); got != "secret" {
			t.Errorf("unexpected secret key header %q", got)
		}
		requestChecked <- struct{}{}

		writer := wsproto.NewWriter(conn)
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeReady, Port: 23456}); err != nil {
			return
		}
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeOpen, StreamID: "stream-1", Data: []byte("hello ")}); err != nil {
			return
		}
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeData, StreamID: "stream-1", Data: []byte("world")}); err != nil {
			return
		}

		for {
			frame, err := wsproto.Receive(conn)
			if err != nil {
				return
			}
			clientFrames <- frame
			if frame.Type == wsproto.TypePing {
				if err := writer.Send(wsproto.Frame{Type: wsproto.TypePong}); err != nil {
					return
				}
			}
		}
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	session, err := Connect(context.Background(), clientcfg.ClientConfig{
		ServerUrl:    host,
		WsUrl:        host,
		SecretKey:    "secret",
		UseLocalHost: true,
	}, "conn-1")
	if err != nil {
		t.Fatalf("connect websocket session: %v", err)
	}
	defer session.Close()

	select {
	case <-requestChecked:
	case <-time.After(time.Second):
		t.Fatal("server did not receive websocket request")
	}
	if session.RemotePort() != 23456 {
		t.Fatalf("expected remote port 23456, got %d", session.RemotePort())
	}

	stream, err := session.Accept()
	if err != nil {
		t.Fatalf("accept tunnel stream: %v", err)
	}
	payload := make([]byte, len("hello world"))
	if _, err := io.ReadFull(stream, payload); err != nil {
		t.Fatalf("read tunnel stream: %v", err)
	}
	if string(payload) != "hello world" {
		t.Fatalf("unexpected stream payload %q", payload)
	}
	if _, err := stream.Write([]byte("response")); err != nil {
		t.Fatalf("write tunnel stream: %v", err)
	}
	select {
	case frame := <-clientFrames:
		if frame.Type != wsproto.TypeData || frame.StreamID != "stream-1" || string(frame.Data) != "response" {
			t.Fatalf("unexpected client frame %#v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not receive stream data")
	}

	if err := session.HealthCheck(time.Second); err != nil {
		t.Fatalf("health check: %v", err)
	}
	select {
	case frame := <-clientFrames:
		if frame.Type != wsproto.TypePing {
			t.Fatalf("expected ping frame, got %#v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not receive health check ping")
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("close tunnel stream: %v", err)
	}
	select {
	case frame := <-clientFrames:
		if frame.Type != wsproto.TypeClose || frame.StreamID != "stream-1" {
			t.Fatalf("expected stream close frame, got %#v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not receive stream close")
	}
}
