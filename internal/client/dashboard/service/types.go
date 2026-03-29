package service

import (
	"time"

	"github.com/amalshaji/portr/internal/client/db"
)

type TunnelSummary struct {
	Subdomain             string    `json:"Subdomain"`
	Localport             int       `json:"Localport"`
	LastRequestID         string    `json:"last_request_id,omitempty"`
	LastMethod            string    `json:"last_method,omitempty"`
	LastURL               string    `json:"last_url,omitempty"`
	LastStatus            int       `json:"last_status,omitempty"`
	LastActivityAt        time.Time `json:"last_activity_at"`
	LastActivityKind      string    `json:"last_activity_kind"`
	HTTPRequestCount      int64     `json:"http_request_count"`
	WebSocketSessionCount int64     `json:"websocket_session_count"`
	ActiveWebSocketCount  int64     `json:"active_websocket_count"`
}

type WebSocketSessionWithEvents struct {
	Session *db.WebSocketSession `json:"session"`
	Events  []db.WebSocketEvent  `json:"events"`
}
