package wstunnel

import (
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

type blockedWriteConn struct {
	net.Conn
	writing chan struct{}
}

func (c *blockedWriteConn) Write(p []byte) (int, error) { close(c.writing); return c.Conn.Write(p) }

func TestSessionCloseInterruptsBlockedDownstreamWrite(t *testing.T) {
	accepted := make(chan *websocket.Conn, 1)
	release := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(c *websocket.Conn) { accepted <- c; <-release }))
	defer server.Close()
	defer close(release)
	peer, err := websocket.Dial("ws"+strings.TrimPrefix(server.URL, "http"), "", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	ws := <-accepted
	sess := &session{writer: wsproto.NewWriter(ws), streams: make(map[string]*streamQueue), closed: make(chan struct{})}
	downstream, remote := net.Pipe()
	defer remote.Close()
	defer downstream.Close()
	blocked := &blockedWriteConn{Conn: downstream, writing: make(chan struct{})}
	finished := make(chan struct{})
	go func() { (&Manager{}).pipeStream(sess, blocked, nil); close(finished) }()
	open, err := wsproto.Receive(peer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = wsproto.Receive(peer); err != nil {
		t.Fatal(err)
	}
	sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: open.StreamID, Data: []byte("response")})
	<-blocked.writing
	close(sess.closed)
	ws.Close()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Error("pipeStream remains blocked after session closure")
	}
}
