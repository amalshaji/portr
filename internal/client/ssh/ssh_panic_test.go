package ssh

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

func TestInstallTransportPublishesCompleteTransport(t *testing.T) {
	listener := &testListener{}
	transport := &tunnelTransport{listener: listener, remotePort: 23456}
	client := &SshClient{}

	if !client.installTransport(context.Background(), transport) {
		t.Fatal("expected transport to be installed")
	}
	if client.transport != transport {
		t.Fatal("expected installed transport to be current")
	}
	if client.ConfigSnapshot().Tunnel.RemotePort != 23456 {
		t.Fatal("expected remote port to publish with transport")
	}
	if err := client.closeTransport(); err != nil {
		t.Fatalf("close transport: %v", err)
	}
	if listener.closeCount != 1 || client.transport != nil {
		t.Fatalf("expected one close and no current transport, closes=%d", listener.closeCount)
	}
}

func TestInstallTransportRejectsShutdownRace(t *testing.T) {
	listener := &testListener{}
	transport := &tunnelTransport{listener: listener}
	client := &SshClient{}
	atomic.StoreInt32(&client.shutdown, 1)

	if client.installTransport(context.Background(), transport) {
		t.Fatal("transport must not publish after shutdown")
	}
	if client.transport != nil || listener.closeCount != 1 {
		t.Fatalf("expected rejected transport to close, current=%v closes=%d", client.transport, listener.closeCount)
	}
}
