package wstunnel

import (
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/tunnel/wsproto"
)

func TestSessionDeliverBackpressuresInsteadOfDroppingFrames(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	stream := sess.addStream("slow-stream")
	for i := 0; i < cap(stream.frames); i++ {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte{byte(i)}})
	}

	delivered := make(chan struct{})
	go func() {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte("last")})
		close(delivered)
	}()

	select {
	case <-delivered:
		t.Fatal("deliver returned while the stream queue was full")
	case <-time.After(25 * time.Millisecond):
	}

	<-stream.frames
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("deliver remained blocked after stream capacity became available")
	}

	if got := len(stream.frames); got != cap(stream.frames) {
		t.Fatalf("expected all frames to remain queued, got %d of %d", got, cap(stream.frames))
	}
}

func TestSessionDeliverUnblocksWhenStreamCloses(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	stream := sess.addStream("closing-stream")
	for i := 0; i < cap(stream.frames); i++ {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "closing-stream"})
	}

	delivered := make(chan struct{})
	go func() {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "closing-stream"})
		close(delivered)
	}()

	sess.removeStream("closing-stream")
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("deliver remained blocked after the stream closed")
	}
}
