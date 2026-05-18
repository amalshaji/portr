package models

import (
	"strings"

	"gorm.io/gorm"
)

const DefaultAddUserEmailSubject = "Welcome to Portr!"

const DefaultAddUserEmailBody = "You have been added to a Portr team. Please set up your account using the temporary password provided."

type InstanceSettings struct {
	Model
	Timestamps
	SMTPEnabled              bool   `gorm:"default:false" json:"smtp_enabled"`
	SMTPHost                 string `gorm:"default:''" json:"smtp_host"`
	SMTPPort                 int    `gorm:"default:587" json:"smtp_port"`
	SMTPUsername             string `gorm:"default:''" json:"smtp_username"`
	SMTPPassword             string `gorm:"default:''" json:"smtp_password"`
	FromAddress              string `gorm:"default:''" json:"from_address"`
	AddUserEmailSubject      string `gorm:"default:'Welcome to Portr!'" json:"add_user_email_subject"`
	AddUserEmailBody         string `gorm:"default:'You have been added to a Portr team. Please set up your account using the temporary password provided.'" json:"add_user_email_body"`
	AutoSignupEnabled        bool   `gorm:"default:false" json:"auto_signup_enabled"`
	AutoSignupAllowedDomains string `gorm:"default:''" json:"auto_signup_allowed_domains"`
	AutoSignupTeamID         *uint  `json:"auto_signup_team_id"`
	AutoSignupTeam           *Team  `gorm:"foreignKey:AutoSignupTeamID" json:"auto_signup_team,omitempty"`
}

func (InstanceSettings) TableName() string {
	return "instance_settings"
}

func DefaultInstanceSettings() InstanceSettings {
	return InstanceSettings{
		SMTPPort:            587,
		AddUserEmailSubject: DefaultAddUserEmailSubject,
		AddUserEmailBody:    DefaultAddUserEmailBody,
	}
}

func GetOrCreateInstanceSettings(db *gorm.DB) (*InstanceSettings, error) {
	settings := DefaultInstanceSettings()
	settings.ID = 1

	err := db.First(&settings, 1).Error
	if err == nil {
		return &settings, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	settings = DefaultInstanceSettings()
	settings.ID = 1
	if err := db.Create(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}

func NormalizeAllowedDomains(domains string) string {
	parts := strings.Split(domains, ",")
	seen := make(map[string]struct{}, len(parts))
	normalized := make([]string, 0, len(parts))

	for _, part := range parts {
		domain := strings.TrimSpace(strings.ToLower(part))
		domain = strings.TrimPrefix(domain, "@")
		domain = strings.TrimSuffix(domain, ".")
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		normalized = append(normalized, domain)
	}

	return strings.Join(normalized, ", ")
}

func AllowedDomainsList(domains string) []string {
	normalized := NormalizeAllowedDomains(domains)
	if normalized == "" {
		return nil
	}

	parts := strings.Split(normalized, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		domain := strings.TrimSpace(part)
		if domain != "" {
			result = append(result, domain)
		}
	}

	return result
}

func EmailMatchesAllowedDomains(email string, domains string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}

	emailDomain := strings.TrimSpace(strings.ToLower(email[at+1:]))
	emailDomain = strings.TrimSuffix(emailDomain, ".")
	if emailDomain == "" {
		return false
	}

	for _, domain := range AllowedDomainsList(domains) {
		if emailDomain == domain {
			return true
		}
	}

	return false
}
