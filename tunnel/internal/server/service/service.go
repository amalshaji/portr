package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/utils"
	"gorm.io/gorm"
)

type Service struct {
	db     *db.Db
	logger *slog.Logger
}

func New(db *db.Db) *Service {
	return &Service{db: db, logger: utils.GetLogger()}
}

func (s *Service) GetReservedConnectionById(ctx context.Context, connectionId string) (*db.Connection, error) {
	var connection db.Connection

	err := s.db.Conn.Preload("CreatedBy").Where("status = 'reserved' AND id = ?", connectionId).First(&connection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("connection not found")
		}
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
	return s.db.Conn.Model(&db.Connection{}).Where("id = ?", connectionId).Update("status", "active").Error
}

func (s *Service) MarkConnectionAsClosed(ctx context.Context, connectionId string) error {
	return s.db.Conn.Model(&db.Connection{}).Where("id = ?", connectionId).Update("status", "closed").Error
}

func (s *Service) GetAllActiveConnections(ctx context.Context) []db.Connection {
	var connections []db.Connection
	s.db.Conn.Where("status = ?", "active").Find(&connections)
	return connections
}
