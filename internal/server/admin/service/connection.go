package service

import (
	"context"

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
		return db.Connection{}, err
	}

	result, err := s.db.Queries.CreateNewConnection(ctx, db.CreateNewConnectionParams{
		Subdomain:    subdomain,
		TeamMemberID: teamUserResult.ID,
	})
	if err != nil {
		return db.Connection{}, err
	}
	return result, nil
}

func (s *Service) MarkConnectionAsClosed(ctx context.Context, connection db.Connection) error {
	return s.db.Queries.MarkConnectionAsClosed(ctx, connection.ID)
}
