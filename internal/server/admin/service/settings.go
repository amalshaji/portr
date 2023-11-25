package service

import (
	"fmt"
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

func (s *Service) UpdateSignupSettings(updateSettingsInput UpdateSettingsInput) (db.Settings, error) {
	if updateSettingsInput.SignupRequiresInvite && updateSettingsInput.AllowRandomUserSignup {
		return db.Settings{}, fmt.Errorf("both signupRequiresInvite and allowRandomUserSignup cannot be true")
	}

	if updateSettingsInput.AllowRandomUserSignup && updateSettingsInput.RandomUserSignupAllowedDomains == "" {
		return db.Settings{}, fmt.Errorf("domains list cannot be empty")
	}

	if !validateAllowedDomains(updateSettingsInput.RandomUserSignupAllowedDomains) {
		return db.Settings{}, fmt.Errorf("domains list must be comma separated and valid")
	}

	var settings db.Settings
	s.db.Conn.First(&settings)

	settings.SignupRequiresInvite = updateSettingsInput.SignupRequiresInvite
	settings.AllowRandomUserSignup = updateSettingsInput.AllowRandomUserSignup
	settings.RandomUserSignupAllowedDomains = updateSettingsInput.RandomUserSignupAllowedDomains

	s.db.Conn.Save(&settings)
	return settings, nil
}
