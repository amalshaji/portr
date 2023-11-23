package service

import (
	"time"

	"github.com/amalshaji/localport/internal/server/db"
)

func (s *Service) ListActiveConnections() []db.Connection {
	var connections []db.Connection
	s.db.Conn.Joins("User").Find(&connections)
	return connections
}

func (s *Service) RegisterNewConnection(subdomain string, secretKey string) (db.Connection, error) {
	var user db.User
	result := s.db.Conn.First(&user, "secret_key = ?", secretKey)
	if result.Error != nil {
		return db.Connection{}, result.Error
	}
	connection := db.Connection{Subdomain: subdomain, User: user}
	result = s.db.Conn.Create(&connection)
	if result.Error != nil {
		return db.Connection{}, result.Error
	}
	return connection, nil
}

func (s *Service) MarkConnectionAsClosed(connection db.Connection) error {
	result := s.db.Conn.Model(&connection).Update("closed_at", time.Now().UTC())
	return result.Error
}
