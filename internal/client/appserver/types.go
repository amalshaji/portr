package appserver

import (
	"time"

	"github.com/amalshaji/portr/internal/constants"
)

const (
	DefaultHost = "127.0.0.1"
	DefaultPort = 7778
)

type StartTunnelRequest struct {
	Name         string                   `json:"name"`
	Type         constants.ConnectionType `json:"type"`
	Host         string                   `json:"host"`
	Port         int                      `json:"port"`
	Subdomain    string                   `json:"subdomain"`
	PoolSize     int                      `json:"pool_size"`
	CallbackURL  string                   `json:"callback_url"`
	CallbackURLs []string                 `json:"callback_urls"`
}

type TunnelStatus struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name,omitempty"`
	Status       string                   `json:"status"`
	Type         constants.ConnectionType `json:"type"`
	Host         string                   `json:"host"`
	Port         int                      `json:"port"`
	Subdomain    string                   `json:"subdomain,omitempty"`
	RemotePort   int                      `json:"remote_port,omitempty"`
	TunnelURL    string                   `json:"tunnel_url,omitempty"`
	CallbackURLs []string                 `json:"callback_urls,omitempty"`
	StartedAt    time.Time                `json:"started_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
	StoppedAt    *time.Time               `json:"stopped_at,omitempty"`
	LastError    string                   `json:"last_error,omitempty"`
}

type TunnelEvent struct {
	ID         string                   `json:"id"`
	TunnelID   string                   `json:"tunnel_id"`
	Type       string                   `json:"type"`
	Name       string                   `json:"name,omitempty"`
	Connection constants.ConnectionType `json:"connection_type"`
	Host       string                   `json:"host"`
	Port       int                      `json:"port"`
	Subdomain  string                   `json:"subdomain,omitempty"`
	RemotePort int                      `json:"remote_port,omitempty"`
	TunnelURL  string                   `json:"tunnel_url,omitempty"`
	Error      string                   `json:"error,omitempty"`
	At         time.Time                `json:"at"`
}
