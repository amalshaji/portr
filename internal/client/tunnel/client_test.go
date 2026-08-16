package tunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

func TestConnectCancelsDuringWebSocketHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- conn
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, connectErr := Connect(ctx, clientcfg.ClientConfig{
			ServerUrl:    listener.Addr().String(),
			WsUrl:        listener.Addr().String(),
			UseLocalHost: true,
		}, "conn-1")
		result <- connectErr
	}()

	var stalledConn net.Conn
	select {
	case stalledConn = <-accepted:
		defer stalledConn.Close()
	case <-time.After(time.Second):
		t.Fatal("websocket client did not begin the handshake")
	}
	cancel()

	select {
	case connectErr := <-result:
		if !errors.Is(connectErr, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", connectErr)
		}
	case <-time.After(250 * time.Millisecond):
		_ = stalledConn.Close()
		t.Fatal("Connect did not stop when its context was canceled")
	}
}

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
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.ProtocolVersion, Port: 23456}); err != nil {
			return
		}
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeOpen, StreamID: "stream-1", Data: []byte("hello ")}); err != nil {
			return
		}
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeWindow, StreamID: "stream-1", Window: wsproto.StreamWindowSize}); err != nil {
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
			if frame.Type == wsproto.TypeWindow {
				continue
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

func TestSaturatedStreamDoesNotBlockOtherStreamsOrHealthChecks(t *testing.T) {
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		writer := wsproto.NewWriter(conn)
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.ProtocolVersion}); err != nil {
			return
		}
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeOpen, StreamID: "slow"}); err != nil {
			return
		}
		for range wsproto.StreamWindowSize + 2 {
			if err := writer.Send(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow", Data: []byte("x")}); err != nil {
				return
			}
		}
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeOpen, StreamID: "fast", Data: []byte("fast")}); err != nil {
			return
		}

		for {
			frame, err := wsproto.Receive(conn)
			if err != nil {
				return
			}
			if frame.Type == wsproto.TypePing {
				_ = writer.Send(wsproto.Frame{Type: wsproto.TypePong})
			}
		}
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	session, err := Connect(context.Background(), clientcfg.ClientConfig{
		ServerUrl:    host,
		WsUrl:        host,
		UseLocalHost: true,
	}, "conn-1")
	if err != nil {
		t.Fatalf("connect websocket session: %v", err)
	}
	defer session.Close()

	slow, err := session.Accept()
	if err != nil {
		t.Fatalf("accept slow stream: %v", err)
	}
	defer slow.Close()

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := session.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()

	var fast net.Conn
	select {
	case fast = <-accepted:
		defer fast.Close()
	case err := <-acceptErr:
		t.Fatalf("accept fast stream: %v", err)
	case <-time.After(time.Second):
		t.Fatal("saturated stream blocked a second stream")
	}
	payload := make([]byte, len("fast"))
	if _, err := io.ReadFull(fast, payload); err != nil {
		t.Fatalf("read fast stream: %v", err)
	}
	if string(payload) != "fast" {
		t.Fatalf("unexpected fast-stream payload %q", payload)
	}
	if err := session.HealthCheck(time.Second); err != nil {
		t.Fatalf("health check behind saturated stream: %v", err)
	}
}

func TestConnectRejectsServerWithoutProtocolVersion(t *testing.T) {
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		// An older server sends a ready frame without a protocol version.
		writer := wsproto.NewWriter(conn)
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeReady}); err != nil {
			return
		}
		_, _ = wsproto.Receive(conn)
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	_, err := Connect(context.Background(), clientcfg.ClientConfig{
		ServerUrl:    host,
		WsUrl:        host,
		SecretKey:    "secret",
		UseLocalHost: true,
	}, "conn-1")
	if err == nil {
		t.Fatal("expected a protocol mismatch error")
	}
	if !strings.Contains(err.Error(), "protocol mismatch") || !strings.Contains(err.Error(), "upgrade the portr server") {
		t.Fatalf("unexpected connect error: %v", err)
	}
}
