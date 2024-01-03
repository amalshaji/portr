package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/oklog/ulid/v2"
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

func (s *Service) RegisterNewHttpConnection(
	ctx context.Context,
	subdomain string,
	secretKey string,
) (db.Connection, error) {
	teamUserResult, err := s.db.Queries.GetTeamUserBySecretKey(ctx, secretKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Connection{}, fmt.Errorf("invalid secret key")
		}
		return db.Connection{}, err
	}

	item, err := s.db.Queries.GetReservedOrActiveConnectionForSubdomain(
		ctx,
		db.GetReservedOrActiveConnectionForSubdomainParams{
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
	if item.ID != "" {
		return db.Connection{}, fmt.Errorf("subdomain is in use")
	}

	result, err := s.db.Queries.CreateNewHttpConnection(ctx, db.CreateNewHttpConnectionParams{
		ID:           ulid.Make().String(),
		Subdomain:    subdomain,
		TeamMemberID: teamUserResult.ID,
		TeamID:       teamUserResult.TeamID,
	})
	if err != nil {
		return db.Connection{}, err
	}
	return result, nil
}

func (s *Service) RegisterNewTcpConnection(
	ctx context.Context,
	secretKey string,
) (db.Connection, error) {
	teamUserResult, err := s.db.Queries.GetTeamUserBySecretKey(ctx, secretKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Connection{}, fmt.Errorf("invalid secret key")
		}
		return db.Connection{}, err
	}

	result, err := s.db.Queries.CreateNewTcpConnection(ctx, db.CreateNewTcpConnectionParams{
		ID:           ulid.Make().String(),
		Port:         nil,
		TeamMemberID: teamUserResult.ID,
		TeamID:       teamUserResult.TeamID,
	})
	if err != nil {
		return db.Connection{}, err
	}
	return result, nil
}

func (s *Service) GetReservedConnectionForSubdomain(
	ctx context.Context,
	subdomain,
	secretKey string,
) (string, error) {
	result, err := s.db.Queries.GetReservedOrActiveConnectionForSubdomain(
		ctx,
		db.GetReservedOrActiveConnectionForSubdomainParams{
			Subdomain: subdomain,
			SecretKey: secretKey,
		})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("unregistered subdomain")
		}
		return "", err
	}
	return result.ID, nil
}

func (s *Service) GetReservedOrActiveConnectionById(
	ctx context.Context,
	id string,
) (db.GetReservedOrActiveConnectionByIdRow, error) {
	result, err := s.db.Queries.GetReservedOrActiveConnectionById(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.GetReservedOrActiveConnectionByIdRow{}, fmt.Errorf("unregistered connection")
		}
		return db.GetReservedOrActiveConnectionByIdRow{}, err
	}
	return result, nil
}

func (s *Service) GetReservedConnectionForPort(
	ctx context.Context,
	port uint32,
	secretKey string,
) (string, error) {
	result, err := s.db.Queries.GetReservedOrActiveConnectionForPort(
		ctx,
		db.GetReservedOrActiveConnectionForPortParams{
			Port:      port,
			SecretKey: secretKey,
		})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("unregistered port")
		}
		return "", err
	}
	return result.ID, nil
}

func (s *Service) MarkConnectionAsClosed(ctx context.Context, connectionId string) error {
	return s.db.Queries.MarkConnectionAsClosed(ctx, connectionId)
}

func (s *Service) MarkConnectionAsActive(ctx context.Context, connectionId string) error {
	return s.db.Queries.MarkConnectionAsActive(ctx, connectionId)
}

func (s *Service) AddPortToConnection(ctx context.Context, connectionId string, port uint32) error {
	return s.db.Queries.AddPortToConnection(ctx, db.AddPortToConnectionParams{
		Port: port,
		ID:   connectionId,
	})
}
