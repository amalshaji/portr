package service

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/amalshaji/portr/internal/server/db/models"
	"github.com/amalshaji/portr/internal/server/smtp"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/valyala/fasttemplate"
)

type CreateTeamInput struct {
	Name string `validate:"required|min_len:4"`
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
		if utils.IsSqliteUniqueConstraintError(err) {
			return nil, errors.New("team name already exists")
		}
		return nil, err
	}

	_, err = s.CreateTeamUser(ctx, userID, team.ID, "admin")
	if err != nil {
		return nil, err
	}

	tx.Commit()
	return &team, nil
}

func (s *Service) sendAddMemberNotification(ctx context.Context, user *db.User, role string, teamId int64, settings *db.GlobalSetting) error {
	// get email template
	team, _ := s.db.Queries.GetTeamById(ctx, teamId)

	context := map[string]interface{}{
		"appUrl":   s.config.AdminUrl(),
		"email":    user.Email,
		"role":     role,
		"teamName": team.Name,
	}

	t := fasttemplate.New(settings.AddMemberEmailSubject.(string), "{{", "}}")
	renderedSubject := t.ExecuteString(context)

	t = fasttemplate.New(settings.AddMemberEmailTemplate.(string), "{{", "}}")
	renderedText := t.ExecuteString(context)

	smtpInput := smtp.SendEmailInput{
		From:    settings.FromAddress.(string),
		To:      user.Email,
		Subject: renderedSubject,
		Body:    renderedText,
	}

	if err := s.smtp.SendEmail(smtpInput, settings); err != nil {
		s.log.Error("failed to send invite notification", "error", err)
		return err
	}

	return nil
}

func (s *Service) AddMember(
	ctx context.Context,
	addMemberInput AddMemberInput,
	addedToTeamId,
	addByTeamUserId int64,
) (*db.User, error) {
	user, err := s.db.Queries.GetUserByEmail(ctx, addMemberInput.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// create user
			newUser, err := s.CreateUser(ctx, GithubUserDetails{
				Email:     addMemberInput.Email,
				AvatarUrl: "",
			}, "", false)
			if err != nil {
				s.log.Error("error while creating user", "error", err)
				return nil, err
			}
			user = *newUser
		} else {
			s.log.Error("error while getting user", "error", err)
			return nil, err
		}
	}

	s.CreateTeamUser(ctx, user.ID, addedToTeamId, addMemberInput.Role)

	go func() {
		settings := s.ListSettings(ctx)
		s.sendAddMemberNotification(ctx, &user, addMemberInput.Role, addedToTeamId, &settings)
	}()

	return &user, nil
}
