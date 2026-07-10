package services

import (
	"context"
	"errors"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/gorm"
)

const activeConnectionSubdomainIndex = "idx_connection_active_subdomain_unique"

type ConnectionService struct {
	db *gorm.DB
}

func NewConnectionService(db *gorm.DB) *ConnectionService {
	return &ConnectionService{db: db}
}

func (s *ConnectionService) Create(ctx context.Context, teamUser *models.TeamUser, connectionType string, subdomain *string) (*models.Connection, error) {
	if connectionType == models.ConnectionTypeHTTP {
		return s.createHTTP(ctx, teamUser, *subdomain)
	}

	connection := models.NewConnection(connectionType, subdomain, teamUser)
	if err := s.db.WithContext(ctx).Create(connection).Error; err != nil {
		return nil, err
	}
	return connection, nil
}

func (s *ConnectionService) createHTTP(ctx context.Context, teamUser *models.TeamUser, subdomain string) (*models.Connection, error) {
	connection := models.NewConnection(models.ConnectionTypeHTTP, &subdomain, teamUser)
	err := withSubdomainRetry(ctx, s.db, func(tx *gorm.DB) error {
		var reservation models.SubdomainReservation
		err := tx.WithContext(ctx).Where("LOWER(subdomain) = ?", subdomain).First(&reservation).Error
		switch {
		case err == nil && reservation.TeamUserID != teamUser.ID:
			return ErrSubdomainReserved
		case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		var existing models.Connection
		err = tx.WithContext(ctx).
			Where("LOWER(subdomain) = ? AND status IN (?, ?)", subdomain, models.ConnectionStatusReserved, models.ConnectionStatusActive).
			First(&existing).Error
		switch {
		case err == nil:
			return ErrSubdomainInUse
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		return tx.WithContext(ctx).Create(connection).Error
	})
	if err != nil {
		if isConstraintError(err, activeConnectionSubdomainIndex) {
			return nil, ErrSubdomainInUse
		}
		return nil, err
	}
	return connection, nil
}
