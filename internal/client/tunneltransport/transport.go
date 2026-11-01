package tunneltransport

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/client/db"
	sshclient "github.com/amalshaji/portr/internal/client/ssh"
	wsclient "github.com/amalshaji/portr/internal/client/tunnel"
	config "github.com/amalshaji/portr/internal/clientconfig"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type Worker interface {
	Start(context.Context) error
	Shutdown(context.Context) error
}

type tunnelSession interface {
	Accept() (net.Conn, error)
	RemotePort() int
	HealthCheck(time.Duration) error
	Close() error
}

type connectSession func(context.Context, config.ClientConfig, string) (tunnelSession, error)

// quicFallbackWindow is how long a server whose QUIC endpoint did not answer is
// reached over websocket before QUIC is tried again.
const quicFallbackWindow = 10 * time.Minute

// quicUnreachableUntil maps a quic_url to the time QUIC may be retried.
var quicUnreachableUntil sync.Map

func connectorFor(transport config.Transport) connectSession {
	if transport == config.TransportQUIC {
		return connectQUICWithFallback(
			func(ctx context.Context, cfg config.ClientConfig, connectionID string) (tunnelSession, error) {
				return wsclient.ConnectQUIC(ctx, cfg, connectionID)
			},
			func(ctx context.Context, cfg config.ClientConfig, connectionID string) (tunnelSession, error) {
				return wsclient.Connect(ctx, cfg, connectionID)
			},
		)
	}
	if transport == config.TransportWebSocket {
		return func(ctx context.Context, cfg config.ClientConfig, connectionID string) (tunnelSession, error) {
			return wsclient.Connect(ctx, cfg, connectionID)
		}
	}
	return func(ctx context.Context, cfg config.ClientConfig, connectionID string) (tunnelSession, error) {
		return sshclient.Connect(ctx, cfg, connectionID)
	}
}

// connectQUICWithFallback uses websocket when UDP to the QUIC endpoint is
// blocked. Errors from a server that answered, such as rejected credentials,
// are returned as they are.
func connectQUICWithFallback(quic, websocket connectSession) connectSession {
	return func(ctx context.Context, cfg config.ClientConfig, connectionID string) (tunnelSession, error) {
		until, skip := quicUnreachableUntil.Load(cfg.QuicUrl)
		if !skip || time.Now().After(until.(time.Time)) {
			session, err := quic(ctx, cfg, connectionID)
			if !errors.Is(err, wsclient.ErrQUICUnreachable) {
				return session, err
			}
			quicUnreachableUntil.Store(cfg.QuicUrl, time.Now().Add(quicFallbackWindow))
			log.Warn("QUIC endpoint unreachable, using websocket", "quic_url", cfg.QuicUrl, "error", err)
		}
		return websocket(ctx, cfg, connectionID)
	}
}

func NewWorker(cfg config.ClientConfig, database *db.Db, tui *tea.Program, fatal func(error), handler func(Event)) Worker {
	client := New(cfg, database, tui, fatal)
	client.SetEventHandler(handler)
	return client
}
