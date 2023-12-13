package service

import (
	"fmt"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/server/smtp"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/valyala/fasttemplate"
)

func (s *Service) sendInviteNotification(invite *db.Invite, teamName string) error {
	// get email template
	settings := s.ListSettings()

	t := fasttemplate.New(settings.UserInviteEmailTemplate, "{{", "}}")
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

func (s *Service) CreateInvite(input CreateInviteInput, invitedBy *db.TeamUser) (*db.Invite, error) {
	email := utils.Trim(input.Email)
	role := utils.Trim(input.Role)

	// check if user exists
	var count int64
	s.db.Conn.Model(&db.TeamUser{}).Joins("User").Where("users.email = ?", email).Count(&count)

	if count == 1 {
		return nil, fmt.Errorf("user is already a member")
	}

	// check if invite exists
	var invite db.Invite
	result := s.db.Conn.
		Where("email = ? AND status = ? AND team_id = ?", email, db.Active, invitedBy.TeamID).
		First(&invite)
	if result.Error == nil {
		return nil, fmt.Errorf("the user is already invited")
	}

	tx := s.db.Conn.Begin()

	// create new invite
	invite = db.Invite{
		Email:             email,
		Role:              db.UserRole(role),
		InvitedByTeamUser: *invitedBy,
		TeamID:            invitedBy.TeamID,
	}

	result = tx.Create(&invite)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}

	// send invite email
	if err := s.sendInviteNotification(&invite, invitedBy.Team.Name); err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return &invite, nil
}

func (s *Service) ListInvites(teamID uint) []db.Invite {
	var invites []db.Invite
	s.db.Conn.Joins("InvitedByTeamUser").Find(&invites, "invites.team_id = ?", teamID)
	return invites
}

var (
	ErrInviteNotFound = fmt.Errorf("invite not found")
)
