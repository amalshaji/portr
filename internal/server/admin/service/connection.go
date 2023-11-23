package service

import (
	"github.com/amalshaji/localport/internal/server/db"
)

func (s *Service) ListActiveConnections() []db.Connection {
	var connections []db.Connection
	s.db.Conn.Joins("User").Find(&connections)
	return connections
}
