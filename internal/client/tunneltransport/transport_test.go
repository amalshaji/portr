package tunneltransport

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	wsclient "github.com/amalshaji/portr/internal/client/tunnel"
	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
)

type fakeSession struct {
	tunnelSession
	name string
}

type countingConnector struct {
	calls   int
	session tunnelSession
	err     error
}

func (c *countingConnector) connect(context.Context, clientcfg.ClientConfig, string) (tunnelSession, error) {
	c.calls++
	return c.session, c.err
}

func TestQUICConnectorFallsBackWhenUnreachable(t *testing.T) {
	quic := &countingConnector{err: fmt.Errorf("%w: blocked", wsclient.ErrQUICUnreachable)}
	websocket := &countingConnector{session: fakeSession{name: "websocket"}}
	connect := connectQUICWithFallback(quic.connect, websocket.connect)
	cfg := clientcfg.ClientConfig{QuicUrl: t.Name() + ":4433"}
	t.Cleanup(func() { quicUnreachableUntil.Delete(cfg.QuicUrl) })

	session, err := connect(context.Background(), cfg, "conn-1")
	if err != nil || session.(fakeSession).name != "websocket" {
		t.Fatalf("session = %v, err = %v; want websocket session", session, err)
	}

	// The unreachable endpoint is remembered, so reconnects skip QUIC.
	if _, err := connect(context.Background(), cfg, "conn-2"); err != nil {
		t.Fatal(err)
	}
	if quic.calls != 1 || websocket.calls != 2 {
		t.Fatalf("quic calls = %d, websocket calls = %d; want 1 and 2", quic.calls, websocket.calls)
	}
}

func TestQUICConnectorRetriesQUICAfterWindow(t *testing.T) {
	quic := &countingConnector{session: fakeSession{name: "quic"}}
	websocket := &countingConnector{session: fakeSession{name: "websocket"}}
	connect := connectQUICWithFallback(quic.connect, websocket.connect)
	cfg := clientcfg.ClientConfig{QuicUrl: t.Name() + ":4433"}
	quicUnreachableUntil.Store(cfg.QuicUrl, time.Now().Add(-time.Second))
	t.Cleanup(func() { quicUnreachableUntil.Delete(cfg.QuicUrl) })

	session, err := connect(context.Background(), cfg, "conn-1")
	if err != nil || session.(fakeSession).name != "quic" {
		t.Fatalf("session = %v, err = %v; want quic session", session, err)
	}
}

func TestQUICConnectorDoesNotFallBackOnRejection(t *testing.T) {
	rejection := errors.New("invalid connection credentials")
	quic := &countingConnector{err: rejection}
	websocket := &countingConnector{session: fakeSession{name: "websocket"}}
	connect := connectQUICWithFallback(quic.connect, websocket.connect)

	_, err := connect(context.Background(), clientcfg.ClientConfig{QuicUrl: t.Name() + ":4433"}, "conn-1")
	if !errors.Is(err, rejection) {
		t.Fatalf("error = %v, want rejection", err)
	}
	if websocket.calls != 0 {
		t.Fatal("rejection fell back to websocket")
	}
}
