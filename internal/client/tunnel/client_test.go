package tunnel

import (
	"context"
	"errors"
	"io"
	"log"
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

// muxServer completes the handshake as a portr server would and hands the
// yamux side of the connection to run.
func muxServer(t *testing.T, port int, run func(*websocket.Conn, *yamux.Session)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		writer := wsproto.NewWriter(conn)
		if err := writer.Send(wsproto.Frame{Type: wsproto.TypeReady, Version: wsproto.ProtocolVersion, Port: port}); err != nil {
			return
		}
		mux, err := yamux.Server(wsproto.MuxConn(conn), wsproto.MuxConfig(discardLogger))
		if err != nil {
			return
		}
		defer mux.Close()
		run(conn, mux)
	}))
}

func connectTo(t *testing.T, server *httptest.Server) *Session {
	t.Helper()
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
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestSessionConnectsAndCarriesStreams(t *testing.T) {
	requestChecked := make(chan struct{}, 1)
	received := make(chan string, 1)
	server := muxServer(t, 23456, func(conn *websocket.Conn, mux *yamux.Session) {
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

		stream, err := mux.OpenStream()
		if err != nil {
			return
		}
		if _, err := stream.Write([]byte("hello world")); err != nil {
			return
		}
		_ = stream.Close()
		reply, _ := io.ReadAll(stream)
		received <- string(reply)
		<-mux.CloseChan()
	})
	defer server.Close()

	session := connectTo(t, server)
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
	payload, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("read tunnel stream: %v", err)
	}
	if string(payload) != "hello world" {
		t.Fatalf("unexpected stream payload %q", payload)
	}
	if _, err := stream.Write([]byte("response")); err != nil {
		t.Fatalf("write tunnel stream: %v", err)
	}
	if err := stream.(interface{ CloseWrite() error }).CloseWrite(); err != nil {
		t.Fatalf("close tunnel stream: %v", err)
	}
	select {
	case got := <-received:
		if got != "response" {
			t.Fatalf("unexpected server payload %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not receive stream data and close")
	}

	if err := session.HealthCheck(time.Second); err != nil {
		t.Fatalf("health check: %v", err)
	}
}

func TestSaturatedStreamDoesNotBlockOtherStreamsOrHealthChecks(t *testing.T) {
	server := muxServer(t, 0, func(_ *websocket.Conn, mux *yamux.Session) {
		slow, err := mux.OpenStream()
		if err != nil {
			return
		}
		// More than one stream window, which the client never reads.
		go func() { _, _ = slow.Write(make([]byte, 4<<20)) }()
		fast, err := mux.OpenStream()
		if err != nil {
			return
		}
		_, _ = fast.Write([]byte("fast"))
		<-mux.CloseChan()
	})
	defer server.Close()

	session := connectTo(t, server)
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

var discardLogger = log.New(io.Discard, "", 0)
