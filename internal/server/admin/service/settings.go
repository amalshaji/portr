package service

import (
	"strings"

	"github.com/amalshaji/localport/internal/server/db"
)

func (s *Service) ListSettings() db.Settings {
	var settings db.Settings
	s.db.Conn.First(&settings)
	return settings
}

func (s *Service) ListSettingsForSignup() db.Settings {
	// Signup page only requires a subset of settings
	var settings db.Settings
	s.db.Conn.Select([]string{"signup_requires_invite", "allow_random_user_signup", "random_user_signup_allowed_domains"}).First(&settings)
	return settings
}

func validateAllowedDomains(domains string) bool {
	allDomains := strings.Split(domains, ",")
	for _, domain := range allDomains {
		if len(strings.Split(domain, ".")) < 2 {
			return false
		}
	}
	return true
}

func (s *Service) UpdateEmailSettings(updateSettingsInput UpdateEmailSettingsInput) (db.Settings, error) {
	var settings db.Settings
	s.db.Conn.First(&settings)

	settings.UserInviteEmailTemplate = updateSettingsInput.UserInviteEmailTemplate

	s.db.Conn.Save(&settings)
	return settings, nil
}
