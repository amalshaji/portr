package wstunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/tunnel/quicproto"
	"github.com/quic-go/quic-go"
	"gorm.io/gorm"
)

type quicTestServer struct {
	manager  *Manager
	addr     string
	pin      string
	database *gorm.DB
	userID   uint
}

func startQUIC(t *testing.T) *quicTestServer {
	t.Helper()
	tunnelService, database := newTestService(t)
	teamUser := serverdb.TeamUser{SecretKey: "secret"}
	if err := database.Create(&teamUser).Error; err != nil {
		t.Fatalf("create team user: %v", err)
	}
	key, err := quicproto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	pin, err := quicproto.KeyPin(key)
	if err != nil {
		t.Fatal(err)
	}
	manager := New(nil, tunnelService)
	addr, err := manager.ListenQUIC("127.0.0.1:0", key)
	if err != nil {
		t.Fatalf("listen quic: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = manager.Shutdown(ctx)
	})
	return &quicTestServer{manager: manager, addr: addr.String(), pin: pin, database: database, userID: teamUser.ID}
}

func (s *quicTestServer) reserve(t *testing.T, id, connectionType, subdomain string) {
	t.Helper()
	connection := serverdb.Connection{ID: id, Type: connectionType, Status: "reserved", CreatedByID: s.userID}
	if subdomain != "" {
		connection.Subdomain = &subdomain
	}
	if err := s.database.Create(&connection).Error; err != nil {
		t.Fatalf("create connection: %v", err)
	}
}

func (s *quicTestServer) dial(t *testing.T) *quic.Conn {
	t.Helper()
	tlsConfig, err := quicproto.ClientTLSConfig(s.pin, false)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := quic.DialAddr(ctx, s.addr, tlsConfig, quicproto.ClientConfig())
	if err != nil {
		t.Fatalf("dial quic: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseWithError(0, "") })
	return conn
}

// hello opens the control stream, sends hello and returns the server's reply.
func hello(t *testing.T, conn *quic.Conn, frame quicproto.Frame) (*quic.Stream, quicproto.Frame, error) {
	t.Helper()
	control, err := conn.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	if err := quicproto.WriteFrame(control, frame); err != nil {
		t.Fatal(err)
	}
	_ = control.SetReadDeadline(time.Now().Add(2 * time.Second))
	reply, err := quicproto.ReadFrame(control)
	_ = control.SetReadDeadline(time.Time{})
	return control, reply, err
}

func validHello(connectionID string) quicproto.Frame {
	return quicproto.Frame{Type: quicproto.TypeHello, ConnectionID: connectionID, SecretKey: "secret", Version: quicproto.ProtocolVersion}
}

// acceptDataStream accepts the next server-opened stream and consumes its
// header byte.
func acceptDataStream(t *testing.T, conn *quic.Conn) net.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		t.Fatalf("accept data stream: %v", err)
	}
	header := make([]byte, 1)
	if _, err := io.ReadFull(stream, header); err != nil || header[0] != quicproto.DataStreamHeader {
		t.Fatalf("data stream header = %v, %v", header, err)
	}
	return quicStream{Stream: stream, conn: conn}
}

func requireCloseReason(t *testing.T, err error, want string) {
	t.Helper()
	var appErr *quic.ApplicationError
	if !errors.As(err, &appErr) || !strings.Contains(appErr.ErrorMessage, want) {
		t.Fatalf("error = %v, want application close containing %q", err, want)
	}
}

func TestQUICServesHTTPThroughStreams(t *testing.T) {
	server := startQUIC(t)
	server.reserve(t, "conn-http", "http", "quic-http")
	conn := server.dial(t)

	_, ready, err := hello(t, conn, validHello("conn-http"))
	if err != nil {
		t.Fatal(err)
	}
	if ready.Type != quicproto.TypeReady || ready.Version != quicproto.ProtocolVersion {
		t.Fatalf("unexpected ready frame %+v", ready)
	}

	visitor := visitorProxy(t, server.manager, "quic-http")
	response := proxyRequest(http.DefaultClient, visitor.URL)
	go serveStreamRequests(acceptDataStream(t, conn), "over quic")
	awaitResponse(t, response, "over quic")
}

func TestQUICTCPTunnelForwardsServerFirstBytesAndHalfClose(t *testing.T) {
	server := startQUIC(t)
	server.reserve(t, "conn-tcp", "tcp", "")
	conn := server.dial(t)

	_, ready, err := hello(t, conn, validHello("conn-tcp"))
	if err != nil {
		t.Fatal(err)
	}
	if ready.Port == 0 {
		t.Fatalf("ready frame has no tcp port: %+v", ready)
	}

	visitor, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(ready.Port))
	if err != nil {
		t.Fatal(err)
	}
	defer visitor.Close()
	_ = visitor.SetDeadline(time.Now().Add(2 * time.Second))

	// The visitor has sent nothing; the stream must still reach the client.
	stream := acceptDataStream(t, conn)
	defer stream.Close()
	if _, err := stream.Write([]byte("greeting")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len("greeting"))
	if _, err := io.ReadFull(visitor, got); err != nil || string(got) != "greeting" {
		t.Fatalf("visitor read %q, %v", got, err)
	}

	if _, err := visitor.Write([]byte("request")); err != nil {
		t.Fatal(err)
	}
	if err := visitor.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	_ = stream.SetReadDeadline(time.Now().Add(2 * time.Second))
	body, err := io.ReadAll(stream)
	if err != nil || string(body) != "request" {
		t.Fatalf("client read %q, %v", body, err)
	}

	// The response still flows after the visitor half-closed.
	if _, err := stream.Write([]byte("response")); err != nil {
		t.Fatal(err)
	}
	_ = stream.(quicStream).CloseWrite()
	tail, err := io.ReadAll(visitor)
	if err != nil || string(tail) != "response" {
		t.Fatalf("visitor read %q, %v", tail, err)
	}
}

func TestQUICAnswersPing(t *testing.T) {
	server := startQUIC(t)
	server.reserve(t, "conn-ping", "http", "quic-ping")
	conn := server.dial(t)
	control, _, err := hello(t, conn, validHello("conn-ping"))
	if err != nil {
		t.Fatal(err)
	}

	if err := quicproto.WriteFrame(control, quicproto.Frame{Type: quicproto.TypePing}); err != nil {
		t.Fatal(err)
	}
	_ = control.SetReadDeadline(time.Now().Add(2 * time.Second))
	pong, err := quicproto.ReadFrame(control)
	if err != nil || pong.Type != quicproto.TypePong {
		t.Fatalf("pong = %+v, %v", pong, err)
	}
}

func TestQUICRejectsBadHello(t *testing.T) {
	tests := []struct {
		name  string
		frame quicproto.Frame
		want  string
	}{
		{name: "wrong secret", frame: quicproto.Frame{Type: quicproto.TypeHello, ConnectionID: "conn-bad", SecretKey: "nope", Version: quicproto.ProtocolVersion}, want: "invalid connection credentials"},
		{name: "unknown connection", frame: validHello("missing"), want: "invalid connection credentials"},
		{name: "missing credentials", frame: quicproto.Frame{Type: quicproto.TypeHello, Version: quicproto.ProtocolVersion}, want: "missing connection credentials"},
		{name: "version mismatch", frame: quicproto.Frame{Type: quicproto.TypeHello, ConnectionID: "conn-bad", SecretKey: "secret", Version: 99}, want: "protocol mismatch"},
	}
	server := startQUIC(t)
	server.reserve(t, "conn-bad", "http", "quic-bad")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := server.dial(t)
			_, _, err := hello(t, conn, tt.frame)
			requireCloseReason(t, err, tt.want)
		})
	}
}

func TestQUICClosesConnectionWithoutHello(t *testing.T) {
	previous := quicHandshakeTimeout
	quicHandshakeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { quicHandshakeTimeout = previous })

	server := startQUIC(t)
	conn := server.dial(t)
	select {
	case <-conn.Context().Done():
		requireCloseReason(t, context.Cause(conn.Context()), "no control stream")
	case <-time.After(2 * time.Second):
		t.Fatal("server kept an unauthenticated connection open")
	}
}

func TestQUICRejectsOtherALPN(t *testing.T) {
	server := startQUIC(t)
	tlsConfig, err := quicproto.ClientTLSConfig(server.pin, false)
	if err != nil {
		t.Fatal(err)
	}
	tlsConfig.NextProtos = []string{"h3"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if conn, err := quic.DialAddr(ctx, server.addr, tlsConfig, quicproto.ClientConfig()); err == nil {
		_ = conn.CloseWithError(0, "")
		t.Fatal("handshake with another ALPN succeeded")
	}
}

func TestQUICShutdownClosesSessions(t *testing.T) {
	server := startQUIC(t)
	server.reserve(t, "conn-shutdown", "http", "quic-shutdown")
	conn := server.dial(t)
	if _, _, err := hello(t, conn, validHello("conn-shutdown")); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.manager.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	select {
	case <-conn.Context().Done():
	case <-time.After(2 * time.Second):
		t.Fatal("client connection stayed open after shutdown")
	}
	if server.manager.HasHTTPBackend("quic-shutdown") {
		t.Fatal("session still registered after shutdown")
	}
}

func TestAttemptLimiterRequiresValidationAboveThreshold(t *testing.T) {
	limiter := &attemptLimiter{}
	for i := 0; i < sourceValidationThreshold; i++ {
		if limiter.overLimit(nil) {
			t.Fatalf("attempt %d required validation", i+1)
		}
	}
	if !limiter.overLimit(nil) {
		t.Fatal("attempt above threshold did not require validation")
	}
}
