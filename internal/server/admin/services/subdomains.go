package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSubdomainReserved      = errors.New("subdomain is reserved")
	ErrSubdomainInUse         = errors.New("subdomain is in use")
	ErrSubdomainUnavailable   = errors.New("subdomain is unavailable")
	ErrReservationExists      = errors.New("reservation already exists")
	ErrReservationLimit       = errors.New("reservation limit reached")
	ErrReservationNotFound    = errors.New("reservation not found")
	ErrReservationUnavailable = errors.New("reservation service is busy")
)

const transactionAttempts = 3

const reservationSubdomainIndex = "idx_subdomain_reservation_name_unique"

type SubdomainClaimStatus string

const (
	SubdomainClaimIdle     SubdomainClaimStatus = "idle"
	SubdomainClaimStarting SubdomainClaimStatus = "starting"
	SubdomainClaimActive   SubdomainClaimStatus = "active"
)

var sqliteSubdomainTransactions sync.Mutex

type ReservedSubdomain struct {
	Reservation models.SubdomainReservation
	ClaimStatus SubdomainClaimStatus
}

type SubdomainService struct {
	db *gorm.DB
}

func NewSubdomainService(db *gorm.DB) *SubdomainService {
	return &SubdomainService{db: db}
}

func (s *SubdomainService) List(ctx context.Context, teamUserID uint) ([]ReservedSubdomain, error) {
	var reservations []models.SubdomainReservation
	if err := s.db.WithContext(ctx).
		Where("team_user_id = ?", teamUserID).
		Order("created_at DESC").
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	if len(reservations) == 0 {
		return []ReservedSubdomain{}, nil
	}

	subdomains := make([]string, 0, len(reservations))
	for _, reservation := range reservations {
		subdomains = append(subdomains, reservation.Subdomain)
	}

	var connections []models.Connection
	if err := s.db.WithContext(ctx).
		Where("LOWER(subdomain) IN ? AND status IN (?, ?)", subdomains, models.ConnectionStatusReserved, models.ConnectionStatusActive).
		Find(&connections).Error; err != nil {
		return nil, err
	}

	statuses := make(map[string]SubdomainClaimStatus, len(connections))
	for _, connection := range connections {
		if connection.Subdomain == nil {
			continue
		}
		key := strings.ToLower(*connection.Subdomain)
		status := claimStatusForConnection(connection.Status)
		if status == SubdomainClaimActive || statuses[key] == "" {
			statuses[key] = status
		}
	}

	items := make([]ReservedSubdomain, 0, len(reservations))
	for _, reservation := range reservations {
		status := statuses[strings.ToLower(reservation.Subdomain)]
		if status == "" {
			status = SubdomainClaimIdle
		}
		items = append(items, ReservedSubdomain{Reservation: reservation, ClaimStatus: status})
	}

	return items, nil
}

func (s *SubdomainService) Reserve(ctx context.Context, teamUserID uint, subdomain string, limit int) (*ReservedSubdomain, error) {
	reservation := &ReservedSubdomain{ClaimStatus: SubdomainClaimIdle}
	err := withSubdomainRetry(ctx, s.db, func(tx *gorm.DB) error {
		reservation.ClaimStatus = SubdomainClaimIdle
		membershipQuery := tx.WithContext(ctx)
		if tx.Dialector.Name() == "postgres" {
			membershipQuery = membershipQuery.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var teamUser models.TeamUser
		if err := membershipQuery.First(&teamUser, teamUserID).Error; err != nil {
			return err
		}

		var existing models.SubdomainReservation
		err := tx.WithContext(ctx).Where("LOWER(subdomain) = ?", subdomain).First(&existing).Error
		switch {
		case err == nil && existing.TeamUserID == teamUserID:
			return ErrReservationExists
		case err == nil:
			return ErrSubdomainUnavailable
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		var openConnection models.Connection
		err = tx.WithContext(ctx).
			Where("LOWER(subdomain) = ? AND status IN (?, ?)", subdomain, models.ConnectionStatusReserved, models.ConnectionStatusActive).
			First(&openConnection).Error
		switch {
		case err == nil && openConnection.CreatedByID != teamUserID:
			return ErrSubdomainUnavailable
		case err == nil:
			reservation.ClaimStatus = claimStatusForConnection(openConnection.Status)
		case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		var count int64
		if err := tx.WithContext(ctx).Model(&models.SubdomainReservation{}).
			Where("team_user_id = ?", teamUserID).
			Count(&count).Error; err != nil {
			return err
		}
		if limit == 0 || count >= int64(limit) {
			return ErrReservationLimit
		}

		created := models.SubdomainReservation{Subdomain: subdomain, TeamUserID: teamUserID}
		if err := tx.WithContext(ctx).Create(&created).Error; err != nil {
			return err
		}
		reservation.Reservation = created
		return nil
	})
	if err != nil {
		if isConstraintError(err, reservationSubdomainIndex) {
			return nil, ErrSubdomainUnavailable
		}
		return nil, err
	}
	return reservation, nil
}

func (s *SubdomainService) Release(ctx context.Context, teamUserID uint, subdomain string) error {
	result := s.db.WithContext(ctx).
		Where("team_user_id = ? AND LOWER(subdomain) = ?", teamUserID, subdomain).
		Delete(&models.SubdomainReservation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrReservationNotFound
	}
	return nil
}

func withSubdomainRetry(ctx context.Context, db *gorm.DB, operation func(*gorm.DB) error) error {
	if db.Dialector.Name() == "sqlite" {
		sqliteSubdomainTransactions.Lock()
		defer sqliteSubdomainTransactions.Unlock()
	}
	for attempt := 0; attempt < transactionAttempts; attempt++ {
		err := db.WithContext(ctx).Transaction(operation, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err == nil || !isRetryableTransactionError(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 10 * time.Millisecond):
		}
	}
	return ErrReservationUnavailable
}

func isConstraintError(err error, constraint string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(constraint))
}

func claimStatusForConnection(status string) SubdomainClaimStatus {
	if status == models.ConnectionStatusActive {
		return SubdomainClaimActive
	}
	return SubdomainClaimStarting
}

func isRetryableTransactionError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "sqlstate 40001") ||
		strings.Contains(message, "could not serialize") ||
		strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "database is busy")
}
