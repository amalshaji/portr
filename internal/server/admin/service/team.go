package service

import (
	"context"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/utils"
)

type CreateTeamInput struct {
	Name string `json:"name"`
}

func (s *Service) CreateTeam(ctx context.Context, createTeamInput CreateTeamInput) (db.Team, error) {
	return s.db.Queries.CreateTeam(ctx, db.CreateTeamParams{
		Name: createTeamInput.Name,
		Slug: utils.Slugify(createTeamInput.Name),
	})
}

func (s *Service) CreateFirstTeam(ctx context.Context, createTeamInput CreateTeamInput, userID int64) (*db.Team, error) {
	tx, _ := s.db.Conn.Begin()
	defer tx.Rollback()

	team, err := s.CreateTeam(ctx, createTeamInput)
	if err != nil {
		return nil, err
	}

	_, err = s.CreateTeamUser(ctx, userID, team.ID, "admin")
	if err != nil {
		return nil, err
	}

	tx.Commit()
	return &team, nil
}
