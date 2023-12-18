package service

import (
	"context"
	"fmt"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/server/smtp"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/valyala/fasttemplate"
)

func (s *Service) sendInviteNotification(ctx context.Context, invite *db.Invite, settings *db.GlobalSetting) error {
	// get email template
	team, _ := s.db.Queries.GetTeamById(ctx, invite.TeamID)

	t := fasttemplate.New(settings.UserInviteEmailSubject.(string), "{{", "}}")
	renderedSubject := t.ExecuteString(map[string]interface{}{
		"appUrl":   s.config.AdminUrl(),
		"email":    invite.Email,
		"role":     invite.Role,
		"teamName": team.Name,
	})

	t = fasttemplate.New(settings.UserInviteEmailTemplate.(string), "{{", "}}")
	renderedText := t.ExecuteString(map[string]interface{}{
		"appUrl":   s.config.AdminUrl(),
		"email":    invite.Email,
		"role":     invite.Role,
		"teamName": team.Name,
	})

	smtpInput := smtp.SendEmailInput{
		From:    settings.FromAddress.(string),
		To:      invite.Email,
		Subject: renderedSubject,
		Body:    renderedText,
	}

	if err := s.smtp.SendEmail(smtpInput, settings); err != nil {
		s.log.Error("failed to send invite notification", "error", err)
		return err
	}

	return nil
}

func (s *Service) CreateInvite(
	ctx context.Context,
	input CreateInviteInput,
	InvitedByTeamMemberID,
	teamID int64,
) (*db.Invite, error) {
	email := utils.Trim(input.Email)
	role := utils.Trim(input.Role)
	// check if user is part of team
	getTeamMemberResult, err := s.db.Queries.GetTeamMemberByEmail(ctx, email)
	if err == nil {
		if getTeamMemberResult.TeamID == teamID {
			return nil, fmt.Errorf("user is already a member")
		}
	}
	// check if invite exists
	count, _ := s.db.Queries.GetNumberOfExistingTeamInvitesForUser(
		ctx,
		db.GetNumberOfExistingTeamInvitesForUserParams{
			Email:  email,
			TeamID: teamID,
		})
	if count > 0 {
		return nil, fmt.Errorf("the user is already invited")
	}

	tx, _ := s.db.Conn.Begin()
	defer tx.Rollback()

	qtx := s.db.Queries.WithTx(tx)

	// check for exising email(user)
	user, err := qtx.GetUserByEmail(ctx, email)

	if err == nil {
		// user exists, create team user
		_, err := s.CreateTeamUser(ctx, user.ID, teamID, role)
		if err != nil {
			return nil, err
		}
		tx.Commit()
		return nil, nil
	}

	invite, err := qtx.CreateInvite(ctx, db.CreateInviteParams{
		Email:                 email,
		Role:                  role,
		Status:                "active",
		InvitedByTeamMemberID: InvitedByTeamMemberID,
		TeamID:                teamID,
	})
	if err != nil {
		return nil, err
	}

	settings := s.ListSettings(ctx)

	// send invite email
	if settings.SmtpEnabled {
		if err := s.sendInviteNotification(ctx, &invite, &settings); err != nil {
			return nil, err
		}
	}
	tx.Commit()
	return &invite, nil
}

func (s *Service) ListInvites(ctx context.Context, teamID int64) []db.GetInvitesForTeamRow {
	invites, _ := s.db.Queries.GetInvitesForTeam(ctx, teamID)
	return invites
}

var (
	ErrInviteNotFound = fmt.Errorf("invite not found")
)
