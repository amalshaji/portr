package tunneltransport

import (
	"context"
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

type EventType string

const (
	EventStarted     EventType = "started"
	EventStopped     EventType = "stopped"
	EventUnhealthy   EventType = "unhealthy"
	EventReconnected EventType = "reconnected"
	EventFailed      EventType = "failed"
)

type Event struct {
	Type       EventType     `json:"type"`
	Tunnel     config.Tunnel `json:"tunnel"`
	TunnelAddr string        `json:"tunnel_addr"`
	Error      string        `json:"error,omitempty"`
	At         time.Time     `json:"at"`
}

func CreateNewConnectionWithContext(ctx context.Context, cfg config.ClientConfig) (string, error) {
	if cfg.Transport == config.TransportWebSocket {
		return wsclient.CreateNewConnectionWithContext(ctx, cfg)
	}
	return sshclient.CreateNewConnectionWithContext(ctx, cfg)
}

func NewWorker(cfg config.ClientConfig, database *db.Db, tui *tea.Program, fatal func(error), handler func(Event)) Worker {
	if cfg.Transport == config.TransportWebSocket {
		client := wsclient.New(cfg, database, tui, fatal)
		if handler != nil {
			client.SetEventHandler(func(event wsclient.Event) {
				handler(Event{
					Type:       EventType(event.Type),
					Tunnel:     event.Tunnel,
					TunnelAddr: event.TunnelAddr,
					Error:      event.Error,
					At:         event.At,
				})
			})
		}
		return client
	}

	client := sshclient.New(cfg, database, tui, fatal)
	if handler != nil {
		client.SetEventHandler(func(event sshclient.Event) {
			handler(Event{
				Type:       EventType(event.Type),
				Tunnel:     event.Tunnel,
				TunnelAddr: event.TunnelAddr,
				Error:      event.Error,
				At:         event.At,
			})
		})
	}
	return client
}
