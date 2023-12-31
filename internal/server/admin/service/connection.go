package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	db "github.com/amalshaji/localport/internal/server/db/models"
)

func (s *Service) ListActiveConnections(ctx context.Context, teamID int64) []db.GetActiveConnectionsForTeamRow {
	result, err := s.db.Queries.GetActiveConnectionsForTeam(ctx, teamID)
	if err != nil {
		s.log.Error("error while fetching active connections", "error", err)
		return []db.GetActiveConnectionsForTeamRow{}
	}
	return result
}

func (s *Service) ListRecentConnections(ctx context.Context, teamID int64) []db.GetRecentConnectionsForTeamRow {
	result, err := s.db.Queries.GetRecentConnectionsForTeam(ctx, teamID)
	if err != nil {
		s.log.Error("error while fetching active connections", "error", err)
		return []db.GetRecentConnectionsForTeamRow{}
	}
	return result
}

func (s *Service) RegisterNewConnection(ctx context.Context, subdomain string, secretKey string) (db.Connection, error) {
	teamUserResult, err := s.db.Queries.GetTeamUserBySecretKey(ctx, secretKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Connection{}, fmt.Errorf("invalid secret key")
		}
		return db.Connection{}, err
	}

	item, err := s.db.Queries.GetActiveConnectionForSubdomain(ctx, db.GetActiveConnectionForSubdomainParams{
		Subdomain: subdomain,
		SecretKey: secretKey,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// do nothing
		} else {
			return db.Connection{}, err
		}
	}

	// do a better check
	if item.ID != 0 {
		return db.Connection{}, fmt.Errorf("subdomain is in use")
	}

	result, err := s.db.Queries.CreateNewConnection(ctx, db.CreateNewConnectionParams{
		Subdomain:    subdomain,
		TeamMemberID: teamUserResult.ID,
		TeamID:       teamUserResult.TeamID,
	})
	if err != nil {
		return db.Connection{}, err
	}
	return result, nil
}

func (s *Service) GetReservedConnectionForSubdomain(ctx context.Context, subdomain, secretKey string) (db.GetActiveConnectionForSubdomainRow, error) {
	result, err := s.db.Queries.GetActiveConnectionForSubdomain(ctx, db.GetActiveConnectionForSubdomainParams{
		Subdomain: subdomain,
		SecretKey: secretKey,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.GetActiveConnectionForSubdomainRow{}, fmt.Errorf("unregistered subdomain")
		}
		return db.GetActiveConnectionForSubdomainRow{}, err
	}
	return result, nil
}

func (s *Service) MarkConnectionAsClosed(ctx context.Context, connectionId int64) error {
	return s.db.Queries.MarkConnectionAsClosed(ctx, connectionId)
}

func (s *Service) MarkConnectionAsActive(ctx context.Context, connectionId int64) error {
	return s.db.Queries.MarkConnectionAsActive(ctx, connectionId)
}
