package service

import (
	"context"

	db "github.com/amalshaji/localport/internal/server/db/models"
)

func (s *Service) ListSettings(ctx context.Context) db.GlobalSetting {
	settings, _ := s.db.Queries.GetGlobalSettings(ctx)
	return settings
}

func (s *Service) UpdateEmailSettings(ctx context.Context, updateSettingsInput UpdateEmailSettingsInput) (db.GlobalSetting, error) {
	err := s.db.Queries.UpdateGlobalSettings(ctx, db.UpdateGlobalSettingsParams{
		UserInviteEmailTemplate: updateSettingsInput.UserInviteEmailTemplate,
		UserInviteEmailSubject:  updateSettingsInput.UserInviteEmailSubject,
	})
	if err != nil {
		return db.GlobalSetting{}, err
	}
	return s.ListSettings(ctx), nil
}
