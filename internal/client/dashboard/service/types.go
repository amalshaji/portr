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
	Status                string    `json:"status"`
}

type WebSocketSessionWithEvents struct {
	Session *db.WebSocketSession `json:"session"`
	Events  []db.WebSocketEvent  `json:"events"`
}

// RequestSummary is a lightweight projection of db.Request used in list
// responses; it excludes headers and bodies, which can be megabytes per row.
type RequestSummary struct {
	ID                 string
	Subdomain          string
	Localport          int
	Host               string
	Url                string
	Method             string
	ResponseStatusCode int
	LoggedAt           time.Time
	IsReplayed         bool
	ParentID           string
	DurationMs         int64
	BytesIn            int64
	BytesOut           int64
	Protocol           string
}

type TunnelStats struct {
	LiveTunnelCount       int64      `json:"live_tunnel_count"`
	HTTPRequestCount      int64      `json:"http_request_count"`
	WebSocketSessionCount int64      `json:"websocket_session_count"`
	ActiveWebSocketCount  int64      `json:"active_websocket_count"`
	LastActivityAt        *time.Time `json:"last_activity_at"`
}

type TunnelsPage struct {
	Tunnels []TunnelSummary
	Total   int64
	Stats   TunnelStats
}
