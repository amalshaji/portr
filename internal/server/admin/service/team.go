package service

import (
	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/utils"
)

func (s *Service) GetTeamCount() int64 {
	var count int64
	s.db.Conn.Model(&db.Team{}).Count(&count)
	return count
}

type CreateTeamInput struct {
	Name string `json:"name"`
}

func (s *Service) CreateTeam(createTeamInput CreateTeamInput) (*db.Team, error) {
	team := db.Team{Name: createTeamInput.Name, Slug: utils.Slugify(createTeamInput.Name)}
	result := s.db.Conn.Create(&team)
	if result.Error != nil {
		return nil, result.Error
	}
	return &team, nil
}

func (s *Service) CreateFirstTeam(createTeamInput CreateTeamInput, user *db.User) (*db.Team, error) {
	tx := s.db.Conn.Begin()

	team, err := s.CreateTeam(createTeamInput)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = s.CreateTeamUser(user, team, db.Admin)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return team, nil
}
