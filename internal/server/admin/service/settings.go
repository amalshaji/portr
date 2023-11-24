package service

import "github.com/amalshaji/localport/internal/server/db"

func (s *Service) ListSettings() []db.Settings {
	var settings []db.Settings
	s.db.Conn.Find(&settings)
	return settings
}

func (s *Service) ListSettingsForSignup() map[string]string {
	// Signup page only requires a subset of settings
	var settings []db.Settings
	s.db.Conn.Find(&settings, "name IN ?", []string{"signup_requires_invite", "allow_random_user_signup", "random_user_signup_allowed_domains"})
	var settingsMap = make(map[string]string)
	for _, setting := range settings {
		settingsMap[setting.Name] = setting.Value
	}
	return settingsMap
}
