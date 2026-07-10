package tunnel

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/tunnel/wsproto"
)

type testListener struct {
	closeCount int
}

func (*testListener) Accept() (net.Conn, error) { return nil, nil }
func (l *testListener) Close() error {
	l.closeCount++
	return nil
}
func (*testListener) Addr() net.Addr { return testAddr("test") }

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

func TestGoSafeReportsPanic(t *testing.T) {
	errCh := make(chan error, 1)
	client := &Client{
		fatal: func(err error) {
			errCh <- err
		},
	}

	client.goSafe("http tunnel", func() {
		panic("boom")
	})

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected panic error, got nil")
		}
		if !strings.Contains(err.Error(), "http tunnel panic: boom") {
			t.Fatalf("unexpected panic error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestHandleAcceptErrorReportsUnexpectedFailure(t *testing.T) {
	listener := &testListener{}
	client := &Client{
		listener: listener,
	}

	err := client.handleAcceptError(listener, errors.New("boom"))
	if err == nil {
		t.Fatal("expected accept error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to accept connection: boom") {
		t.Fatalf("unexpected accept error: %v", err)
	}
}

func TestHandleAcceptErrorIgnoresReplacedListener(t *testing.T) {
	oldListener := &testListener{}
	client := &Client{
		listener: &testListener{},
	}

	err := client.handleAcceptError(oldListener, net.ErrClosed)
	if !errors.Is(err, errClientShuttingDown) {
		t.Fatalf("expected shutdown sentinel, got %v", err)
	}
}

func TestCloseListenerIfCurrentDoesNotCloseReplacement(t *testing.T) {
	oldListener := &testListener{}
	newListener := &testListener{}
	client := &Client{
		listener: newListener,
	}

	client.closeListenerIfCurrent(oldListener)

	if oldListener.closeCount != 0 {
		t.Fatalf("expected old listener not to be closed, got %d closes", oldListener.closeCount)
	}
	if newListener.closeCount != 0 {
		t.Fatalf("expected replacement listener not to be closed, got %d closes", newListener.closeCount)
	}
	if client.listener != newListener {
		t.Fatal("expected replacement listener to remain active")
	}

	client.closeListenerIfCurrent(newListener)

	if newListener.closeCount != 1 {
		t.Fatalf("expected current listener to be closed once, got %d closes", newListener.closeCount)
	}
	if client.listener != nil {
		t.Fatal("expected current listener to be cleared")
	}
}

func TestForwardListenerErrorsReportsFatal(t *testing.T) {
	errCh := make(chan error, 1)
	fatalCh := make(chan error, 1)
	client := &Client{
		fatal: func(err error) {
			fatalCh <- err
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expected := errors.New("listener failed")
	client.forwardListenerErrors(ctx, errCh)
	errCh <- expected

	select {
	case err := <-fatalCh:
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fatal listener error")
	}
}

func TestForwardListenerErrorsIgnoresShutdownError(t *testing.T) {
	errCh := make(chan error, 1)
	fatalCh := make(chan error, 1)
	client := &Client{
		fatal: func(err error) {
			fatalCh <- err
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client.forwardListenerErrors(ctx, errCh)
	errCh <- errClientShuttingDown

	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDeliverTunnelFrameIgnoresClosedTunnelStream(t *testing.T) {
	client := &Client{
		streams: make(map[string]*tunnelStream),
	}
	stream := client.addTunnelStream("stream-1")
	stream.close()

	client.deliverTunnelFrame(wsproto.Frame{
		Type:     wsproto.TypeData,
		StreamID: "stream-1",
		Data:     []byte("late frame"),
	})

	select {
	case frame := <-stream.frames:
		t.Fatalf("expected closed stream to ignore late frame, got %#v", frame)
	default:
	}
}

func TestCloseTunnelStreamsUnblocksStreamConnRead(t *testing.T) {
	client := &Client{
		streams: make(map[string]*tunnelStream),
	}
	stream := client.addTunnelStream("stream-1")
	conn := newTunnelStreamConn("stream-1", nil, stream.frames, stream.done, func(wsproto.Frame) error {
		return nil
	})

	client.closeTunnelStreams()

	done := make(chan error, 1)
	go func() {
		_, err := conn.Read(make([]byte, 1))
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("expected stream read to unblock with net.ErrClosed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream read to unblock")
	}
}
