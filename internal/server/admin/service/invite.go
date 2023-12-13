package service

import (
	"context"
	"fmt"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/server/smtp"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/valyala/fasttemplate"
)

func (s *Service) sendInviteNotification(ctx context.Context, invite *db.Invite, teamName string) error {
	// get email template
	settings := s.ListSettings(ctx)

	t := fasttemplate.New(settings.UserInviteEmailTemplate.(string), "{{", "}}")
	renderedText := t.ExecuteString(map[string]interface{}{
		"appUrl":   s.config.AdminUrl(),
		"email":    invite.Email,
		"role":     invite.Role,
		"teamName": teamName,
	})

	smtpInput := smtp.SendEmailInput{
		From:    s.config.Admin.Smtp.FromEmail,
		To:      invite.Email,
		Subject: "Invitation to join Localport",
		Body:    renderedText,
	}

	if err := s.smtp.SendEmail(smtpInput); err != nil {
		s.log.Error("failed to send invite notification", "error", err)
		return err
	}
	return nil
}

func (s *Service) CreateInvite(ctx context.Context, input CreateInviteInput, InvitedByTeamMemberID, teamID int64) (*db.Invite, error) {
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

	team, _ := qtx.GetTeamById(ctx, teamID)

	// send invite email
	if err := s.sendInviteNotification(ctx, &invite, team.Name); err != nil {
		return nil, err
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
