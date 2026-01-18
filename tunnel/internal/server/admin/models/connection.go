package models

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// Connection represents a tunnel connection
// This should match the existing schema used by the tunnel server
type Connection struct {
	ID          string     `gorm:"primarykey" json:"id"`
	Type        string     `json:"type"`
	Subdomain   *string    `json:"subdomain"`
	Port        *uint32    `json:"port"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	CreatedByID uint       `json:"created_by_id"`
	CreatedBy   TeamUser   `json:"created_by,omitempty"`
	TeamID      uint       `json:"team_id"`
	Team        Team       `json:"team,omitempty"`
}

func (Connection) TableName() string {
	return "connection"
}

// Connection types
const (
	ConnectionTypeHTTP = "http"
	ConnectionTypeTCP  = "tcp"
)

// Connection statuses
const (
	ConnectionStatusReserved = "reserved"
	ConnectionStatusActive   = "active"
	ConnectionStatusClosed   = "closed"
)

// GenerateConnectionID generates a new ULID for connection
func GenerateConnectionID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

// NewConnection creates a new connection
func NewConnection(connType string, subdomain *string, createdBy *TeamUser) *Connection {
	conn := &Connection{
		ID:          GenerateConnectionID(),
		Type:        connType,
		Status:      ConnectionStatusReserved,
		CreatedByID: createdBy.ID,
		TeamID:      createdBy.TeamID,
	}

	if connType == ConnectionTypeHTTP && subdomain != nil {
		conn.Subdomain = subdomain
	}

	return conn
}

// IsActive checks if the connection is active
func (c *Connection) IsActive() bool {
	return c.Status == ConnectionStatusActive
}

// Duration returns the duration of the connection
func (c *Connection) Duration() *time.Duration {
	if c.StartedAt == nil {
		return nil
	}

	end := time.Now()
	if c.ClosedAt != nil {
		end = *c.ClosedAt
	}

	duration := end.Sub(*c.StartedAt)
	return &duration
}
