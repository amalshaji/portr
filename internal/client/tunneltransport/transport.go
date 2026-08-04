package tunneltransport

import (
	"context"
	"net"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	sshclient "github.com/amalshaji/portr/internal/client/ssh"
	wsclient "github.com/amalshaji/portr/internal/client/tunnel"
	tea "github.com/charmbracelet/bubbletea"
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

func connectorFor(transport config.Transport) connectSession {
	if transport == config.TransportWebSocket {
		return func(ctx context.Context, cfg config.ClientConfig, connectionID string) (tunnelSession, error) {
			return wsclient.Connect(ctx, cfg, connectionID)
		}
	}
	return func(ctx context.Context, cfg config.ClientConfig, connectionID string) (tunnelSession, error) {
		return sshclient.Connect(ctx, cfg, connectionID)
	}
}

func NewWorker(cfg config.ClientConfig, database *db.Db, tui *tea.Program, fatal func(error), handler func(Event)) Worker {
	client := New(cfg, database, tui, fatal)
	client.SetEventHandler(handler)
	return client
}
