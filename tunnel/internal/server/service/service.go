package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/amalshaji/portr/internal/server/db"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

type Service struct {
	db *db.Db
}

func New(db *db.Db) *Service {
	return &Service{db: db}
}

func (s *Service) GetReservedConnectionById(ctx context.Context, connectionId string) (*db.Connection, error) {
	var connection db.Connection

	err := s.db.Conn.Preload("CreatedBy").Where("status = 'reserved' AND id = ?", connectionId).First(&connection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("Connection not found", "connection_id", connectionId)
			return nil, fmt.Errorf("connection not found")
		}
		log.Error("Failed to get reserved connection", "error", err)
		return nil, err
	}

	return &connection, nil
}

func (s *Service) AddPortToConnection(ctx context.Context, connectionId string, port uint32) error {
	connection, err := s.GetReservedConnectionById(ctx, fmt.Sprint(connectionId))
	if err != nil {
		return err
	}
	connection.Port = &port
	return s.db.Conn.Save(connection).Error
}

func (s *Service) MarkConnectionAsActive(ctx context.Context, connectionId string) error {
	return s.db.Conn.Model(&db.Connection{}).
		Where("id = ?", connectionId).
		Updates(map[string]any{"status": "active", "started_at": time.Now().UTC()}).Error
}

func (s *Service) MarkConnectionAsClosed(ctx context.Context, connectionId string) error {
	return s.db.Conn.Model(&db.Connection{}).
		Where("id = ?", connectionId).
		Updates(map[string]any{"status": "closed", "closed_at": time.Now().UTC()}).Error
}

func (s *Service) GetAllActiveConnections(ctx context.Context) []db.Connection {
	var connections []db.Connection
	s.db.Conn.Where("status = ?", "active").Find(&connections)
	return connections
}
