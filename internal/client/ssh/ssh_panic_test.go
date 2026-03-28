package ssh

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

type testListener struct{}

func (testListener) Accept() (net.Conn, error) { return nil, nil }
func (testListener) Close() error              { return nil }
func (testListener) Addr() net.Addr            { return testAddr("test") }

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

func TestGoSafeReportsPanic(t *testing.T) {
	errCh := make(chan error, 1)
	client := &SshClient{
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
	client := &SshClient{
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
	client := &SshClient{
		listener: &testListener{},
	}

	err := client.handleAcceptError(oldListener, net.ErrClosed)
	if !errors.Is(err, errClientShuttingDown) {
		t.Fatalf("expected shutdown sentinel, got %v", err)
	}
}

func TestForwardListenerErrorsReportsFatal(t *testing.T) {
	errCh := make(chan error, 1)
	fatalCh := make(chan error, 1)
	client := &SshClient{
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
	client := &SshClient{
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
