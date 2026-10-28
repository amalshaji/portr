package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

// legacyServer behaves like a portr server released before yamux: it rejects
// any protocol version other than 1 with its mismatch error.
func legacyServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		if version := conn.Request().Header.Get(wsproto.ProtocolVersionHeader); version != "1" {
			_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: fmt.Sprintf(
				"portr websocket protocol mismatch: server speaks v1, client sent %q; upgrade the portr client", version)})
			_ = conn.Close()
			return
		}
		handler(conn)
	}))
}

func TestConnectFallsBackToV1AndRemembersTheServer(t *testing.T) {
	var attempts, v2Attempts atomic.Int32
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		attempts.Add(1)
		if conn.Request().Header.Get(wsproto.ProtocolVersionHeader) != "1" {
			v2Attempts.Add(1)
			_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: "portr websocket protocol mismatch: server speaks v1, client sent \"2\"; upgrade the portr client"})
			return
		}
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.LegacyProtocolVersion, Port: 4242})
		_, _ = wsproto.Receive(conn)
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")
	cfg := clientcfg.ClientConfig{ServerUrl: host, WsUrl: host, UseLocalHost: true}

	for range 2 {
		session, err := Connect(context.Background(), cfg, "conn-1")
		if err != nil {
			t.Fatalf("connect to a v1 server: %v", err)
		}
		if session.legacy == nil || session.RemotePort() != 4242 {
			t.Fatalf("expected a v1 session on port 4242, got legacy=%v port=%d", session.legacy != nil, session.RemotePort())
		}
		_ = session.Close()
	}
	if got := v2Attempts.Load(); got != 1 {
		t.Fatalf("expected one v2 attempt before remembering the server, got %d", got)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 handshakes (v2, v1, v1), got %d", got)
	}
}

func TestConnectDoesNotFallBackOnOtherErrors(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		attempts.Add(1)
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: "invalid connection credentials"})
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")
	_, err := Connect(context.Background(), clientcfg.ClientConfig{ServerUrl: host, WsUrl: host, UseLocalHost: true}, "conn-1")
	if err == nil || !strings.Contains(err.Error(), "invalid connection credentials") {
		t.Fatalf("expected the credentials error, got %v", err)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("expected no fallback attempt, got %d handshakes", got)
	}
}

func TestLegacySessionConnectsAndCarriesStreams(t *testing.T) {
	requestChecked := make(chan struct{}, 1)
	clientFrames := make(chan wsproto.Frame, 4)
	server := legacyServer(t, func(conn *websocket.Conn) {
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
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.LegacyProtocolVersion, Port: 23456}); err != nil {
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
	})
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

func TestLegacySaturatedStreamDoesNotBlockOtherStreamsOrHealthChecks(t *testing.T) {
	server := legacyServer(t, func(conn *websocket.Conn) {
		writer := wsproto.NewWriter(conn)
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.LegacyProtocolVersion}); err != nil {
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
	})
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

func TestLegacyHealthTimeoutIncludesBlockedWriter(t *testing.T) {
	release := make(chan struct{})
	server := legacyServer(t, func(c *websocket.Conn) {
		wsproto.NewWriter(c).Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.LegacyProtocolVersion})
		<-release
	})
	defer server.Close()
	defer close(release)
	host := strings.TrimPrefix(server.URL, "http://")
	s, err := Connect(context.Background(), clientcfg.ClientConfig{ServerUrl: host, WsUrl: host, UseLocalHost: true}, "conn")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.conn.SetDeadline(time.Now()); _ = s.Close() }()
	started := make(chan struct{})
	go func() {
		close(started)
		for range 512 {
			if s.legacy.writer.Send(wsproto.Frame{Type: wsproto.TypeData, Data: make([]byte, 32*1024)}) != nil {
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

func TestLegacyStreamReportsMissingDeadlineSupport(t *testing.T) {
	// net/http needs working read deadlines to hand a connection over for a
	// websocket upgrade; callers wrap streams that report none.
	conn := newTunnelStreamConn("stream", nil, nil, nil, nil, nil)
	if err := conn.SetReadDeadline(time.Now()); err == nil {
		t.Fatal("expected v1 streams to report that deadlines are unsupported")
	}
}
