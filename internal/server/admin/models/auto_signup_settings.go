package models

import (
	"strings"

	"gorm.io/gorm"
)

type AutoSignupSettings struct {
	Model
	Timestamps
	AutoSignupEnabled bool `gorm:"default:false" json:"auto_signup_enabled"`
}

type AutoSignupDomain struct {
	Model
	Timestamps
	Domain string `gorm:"uniqueIndex;not null" json:"domain"`
	TeamID uint   `gorm:"not null;index" json:"team_id"`
	Team   Team   `gorm:"constraint:OnDelete:CASCADE;" json:"team,omitempty"`
}

func (AutoSignupSettings) TableName() string {
	return "auto_signup_settings"
}

func (AutoSignupDomain) TableName() string {
	return "auto_signup_domains"
}

func DefaultAutoSignupSettings() AutoSignupSettings {
	return AutoSignupSettings{}
}

func GetOrCreateAutoSignupSettings(db *gorm.DB) (*AutoSignupSettings, error) {
	settings := DefaultAutoSignupSettings()
	settings.ID = 1

	if err := db.Where("id = ?", uint(1)).FirstOrCreate(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}

func NormalizeAutoSignupDomain(domain string) (string, bool) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimPrefix(domain, "@")
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" {
		return "", false
	}
	if strings.Contains(domain, "@") || strings.Contains(domain, ",") {
		return "", false
	}
	if strings.ContainsAny(domain, " \t\r\n") {
		return "", false
	}
	if strings.HasPrefix(domain, ".") || strings.Contains(domain, "..") {
		return "", false
	}

	return domain, true
}

func EmailDomain(email string) (string, bool) {
	email = strings.TrimSpace(strings.ToLower(email))
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return "", false
	}

	return NormalizeAutoSignupDomain(email[at+1:])
}
