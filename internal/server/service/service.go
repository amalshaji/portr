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
	err := s.db.Conn.WithContext(ctx).Preload("CreatedBy").Where("id = ?", connectionId).First(&connection).Error
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

func (s *Service) MarkConnectionAsActive(ctx context.Context, connectionId string) error {
	return s.activateConnection(ctx, connectionId, nil)
}

func (s *Service) MarkTCPConnectionAsActive(ctx context.Context, connectionId string, port uint32) error {
	return s.activateConnection(ctx, connectionId, &port)
}

func (s *Service) activateConnection(ctx context.Context, connectionId string, port *uint32) error {
	updates := map[string]any{"status": "active", "started_at": time.Now().UTC(), "closed_at": nil}
	if port != nil {
		updates["port"] = *port
	}
	return s.db.Conn.WithContext(ctx).Model(&db.Connection{}).
		Where("id = ?", connectionId).
		Updates(updates).Error
}

func (s *Service) MarkConnectionAsClosed(ctx context.Context, connectionId string) error {
	return s.db.Conn.WithContext(ctx).Model(&db.Connection{}).
		Where("id = ?", connectionId).
		Updates(map[string]any{"status": "closed", "closed_at": time.Now().UTC()}).Error
}

func (s *Service) CloseAllActiveConnections(ctx context.Context) error {
	return s.db.Conn.WithContext(ctx).Model(&db.Connection{}).
		Where("status = ?", "active").
		Updates(map[string]any{"status": "closed", "closed_at": time.Now().UTC()}).Error
}

func (s *Service) GetAllActiveConnections(ctx context.Context) ([]db.Connection, error) {
	var connections []db.Connection
	result := s.db.Conn.WithContext(ctx).Where("status = ?", "active").Find(&connections)
	return connections, result.Error
}
