package models

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

type Team struct {
	Model
	Timestamps
	Name      string     `gorm:"uniqueIndex;not null" json:"name"`
	Slug      string     `gorm:"uniqueIndex;not null" json:"slug"`
	Users     []User     `gorm:"many2many:team_users;" json:"users,omitempty"`
	TeamUsers []TeamUser `json:"team_users,omitempty"`
}

func (Team) TableName() string {
	return "team"
}

func (t *Team) BeforeCreate(tx *gorm.DB) error {
	if t.Slug == "" {
		t.Slug = makeSlug(t.Name)
	}
	return nil
}

func makeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")

	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	s = reg.ReplaceAllString(s, "")

	reg = regexp.MustCompile(`-+`)
	s = reg.ReplaceAllString(s, "-")

	s = strings.Trim(s, "-")

	return s
}

type TeamUser struct {
	Model
	Timestamps
	SecretKey string `gorm:"uniqueIndex;not null" json:"secret_key"`
	Role      string `gorm:"default:'member'" json:"role"`
	TeamID    uint   `json:"team_id"`
	Team      Team   `json:"team,omitempty"`
	UserID    uint   `json:"user_id"`
	User      User   `json:"user,omitempty"`
}

func (TeamUser) TableName() string {
	return "team_users"
}

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

func (tu *TeamUser) BeforeCreate(tx *gorm.DB) error {
	if tu.SecretKey == "" {
		tu.SecretKey = GenerateSecretKey()
	}
	return nil
}

func GenerateSecretKey() string {
	bytes := make([]byte, 21)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return "portr_" + hex.EncodeToString(bytes)
}

func (tu *TeamUser) IsAdmin() bool {
	return tu.Role == RoleAdmin
}

func (tu *TeamUser) CanManageTeam() bool {
	return tu.IsAdmin() || tu.User.IsSuperuser
}
