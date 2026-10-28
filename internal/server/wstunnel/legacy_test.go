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

func TestLegacySessionDeliverKeepsOtherStreamsResponsiveWhenOneIsFull(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	slow := sess.addStream("slow-stream")
	fast := sess.addStream("fast-stream")
	for i := 0; i < cap(slow.frames); i++ {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte{byte(i)}})
	}
	sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte("overflow")})
	sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "fast-stream", Data: []byte("ready")})
	select {
	case frame := <-fast.frames:
		if string(frame.Data) != "ready" {
			t.Fatalf("unexpected fast-stream frame %q", frame.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("full stream blocked delivery to another stream")
	}

	sess.streamMu.Lock()
	_, slowPresent := sess.streams["slow-stream"]
	sess.streamMu.Unlock()
	if slowPresent {
		t.Fatal("expected overflowing stream to be removed")
	}
}

func TestLegacyStreamCreditWaitUnblocksWhenStreamCloses(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	stream := sess.addStream("closing-stream")

	waiting := make(chan bool, 1)
	go func() {
		waiting <- stream.takeCredit(sess.closed)
	}()

	sess.removeStream("closing-stream")
	select {
	case acquired := <-waiting:
		if acquired {
			t.Fatal("credit wait succeeded after stream close")
		}
	case <-time.After(time.Second):
		t.Fatal("credit wait remained blocked after the stream closed")
	}
}

func TestLegacySessionCloseInterruptsBlockedDownstreamWrite(t *testing.T) {
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
