package tunneltransport

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
)

type blockingSession struct {
	remotePort int
	done       chan struct{}
	closeOnce  sync.Once
}

func newBlockingSession(remotePort int) *blockingSession {
	return &blockingSession{remotePort: remotePort, done: make(chan struct{})}
}

func (s *blockingSession) Accept() (net.Conn, error) {
	<-s.done
	return nil, net.ErrClosed
}

func (s *blockingSession) RemotePort() int { return s.remotePort }
func (s *blockingSession) HealthCheck(time.Duration) error {
	return nil
}
func (s *blockingSession) Close() error {
	s.closeOnce.Do(func() { close(s.done) })
	return nil
}

func TestClientRunsAndShutsDownTransportSession(t *testing.T) {
	session := newBlockingSession(23456)
	client := New(clientcfg.ClientConfig{
		ConnectionID:        "conn-1",
		DisableTerminalLogs: true,
		TunnelUrl:           "example.test",
		Tunnel: clientcfg.Tunnel{
			Name: "demo",
			Type: constants.Tcp,
			Host: "127.0.0.1",
			Port: 3000,
		},
	}, nil, nil, nil)
	client.connect = func(ctx context.Context, cfg clientcfg.ClientConfig, connectionID string) (tunnelSession, error) {
		if connectionID != "conn-1" {
			t.Fatalf("unexpected connection ID %q", connectionID)
		}
		return session, nil
	}

	events := make(chan Event, 2)
	client.SetEventHandler(func(event Event) { events <- event })
	startDone := make(chan error, 1)
	go func() { startDone <- client.Start(context.Background()) }()

	select {
	case event := <-events:
		if event.Type != EventStarted {
			t.Fatalf("expected started event, got %q", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for transport startup")
	}
	if got := client.ConfigSnapshot().Tunnel.RemotePort; got != 23456 {
		t.Fatalf("expected remote port 23456, got %d", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown client: %v", err)
	}
	if err := <-startDone; err != nil {
		t.Fatalf("start returned an error during shutdown: %v", err)
	}
	select {
	case event := <-events:
		if event.Type != EventStopped {
			t.Fatalf("expected stopped event, got %q", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stopped event")
	}
}
