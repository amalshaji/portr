package wstunnel

import (
	"io"
	"log"
	"net"
	"testing"
	"time"

	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/hashicorp/yamux"
)

type blockedWriteConn struct {
	net.Conn
	writing chan struct{}
}

func (c *blockedWriteConn) Write(p []byte) (int, error) { close(c.writing); return c.Conn.Write(p) }

// newMuxSession returns a ready server session and the client end of its yamux
// session, joined by an in-memory pipe.
func newMuxSession(t *testing.T) (*session, *yamux.Session) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	serverMux, err := yamux.Server(serverConn, wsproto.MuxConfig(discardLogger))
	if err != nil {
		t.Fatal(err)
	}
	clientMux, err := yamux.Client(clientConn, wsproto.MuxConfig(discardLogger))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverMux.Close(); _ = clientMux.Close() })
	sess := &session{mux: serverMux, ready: make(chan struct{}), closed: make(chan struct{})}
	close(sess.ready)
	return sess, clientMux
}

func tcpPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := listener.Accept()
		accepted <- conn
	}()
	dialed, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	server := <-accepted
	t.Cleanup(func() { _ = dialed.Close(); _ = server.Close() })
	return dialed, server
}

func TestPipeStreamCarriesBothDirectionsWithHalfClose(t *testing.T) {
	sess, clientMux := newMuxSession(t)
	visitor, proxied := tcpPair(t)

	finished := make(chan struct{})
	go func() { (&Manager{}).pipeStream(sess, proxied, []byte("head ")); close(finished) }()

	stream, err := clientMux.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := visitor.Write([]byte("body")); err != nil {
		t.Fatal(err)
	}
	if err := visitor.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	request, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if string(request) != "head body" {
		t.Fatalf("unexpected request bytes %q", request)
	}

	if _, err := stream.Write([]byte("response")); err != nil {
		t.Fatal(err)
	}
	_ = stream.Close()
	response, err := io.ReadAll(visitor)
	if err != nil {
		t.Fatal(err)
	}
	if string(response) != "response" {
		t.Fatalf("unexpected response bytes %q", response)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("pipeStream did not finish after both directions closed")
	}
}

func TestPipeStreamWaitsForReadyFrame(t *testing.T) {
	sess, clientMux := newMuxSession(t)
	sess.ready = make(chan struct{})
	_, proxied := tcpPair(t)
	go (&Manager{}).pipeStream(sess, proxied, nil)

	accepted := make(chan struct{})
	go func() {
		if _, err := clientMux.AcceptStream(); err == nil {
			close(accepted)
		}
	}()
	select {
	case <-accepted:
		t.Fatal("stream opened before the ready frame was sent")
	case <-time.After(50 * time.Millisecond):
	}
	close(sess.ready)
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("stream did not open after the ready frame")
	}
	close(sess.closed)
}

func TestSessionCloseInterruptsBlockedDownstreamWrite(t *testing.T) {
	sess, clientMux := newMuxSession(t)
	downstream, remote := net.Pipe()
	defer remote.Close()
	defer downstream.Close()
	blocked := &blockedWriteConn{Conn: downstream, writing: make(chan struct{})}
	finished := make(chan struct{})
	go func() { (&Manager{}).pipeStream(sess, blocked, nil); close(finished) }()

	stream, err := clientMux.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Write([]byte("response")); err != nil {
		t.Fatal(err)
	}
	<-blocked.writing
	close(sess.closed)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Error("pipeStream remains blocked after session closure")
	}
}

var discardLogger = log.New(io.Discard, "", 0)

func TestPipeStreamStopsWaitingForVisitorAfterLingerTimeout(t *testing.T) {
	previous := streamLingerTimeout
	streamLingerTimeout = 50 * time.Millisecond
	t.Cleanup(func() { streamLingerTimeout = previous })

	sess, clientMux := newMuxSession(t)
	sess.connection = &serverdb.Connection{ID: "conn-linger", Type: "http"}
	_, proxied := tcpPair(t)
	finished := make(chan struct{})
	go func() { (&Manager{}).pipeStream(sess, proxied, nil); close(finished) }()

	stream, err := clientMux.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = stream.Write([]byte("response"))
	_ = stream.Close()
	// The visitor never closes its side.
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("pipeStream kept waiting on a visitor that never closed")
	}
}
