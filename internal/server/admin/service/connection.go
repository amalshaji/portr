package service

import (
	"time"

	"github.com/amalshaji/localport/internal/server/db"
)

func (s *Service) ListActiveConnections(teamID uint) []db.Connection {
	var connections []db.Connection
	s.db.Conn.
		Limit(20).
		Order("connections.id desc").
		Joins("User").
		Find(&connections, "closed_at IS NULL AND team_id = ?", teamID)
	return connections
}

func (s *Service) ListRecentConnections(teamID uint) []db.Connection {
	var connections []db.Connection
	s.db.Conn.Limit(20).Order("connections.id desc").Joins("User").Find(&connections, "team_id = ?", teamID)
	return connections
}

func (s *Service) RegisterNewConnection(subdomain string, secretKey string) (db.Connection, error) {
	var teamUser db.TeamUser
	result := s.db.Conn.First(&teamUser, "secret_key = ?", secretKey)
	if result.Error != nil {
		return db.Connection{}, result.Error
	}
	connection := db.Connection{Subdomain: subdomain, TeamUser: teamUser}
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
