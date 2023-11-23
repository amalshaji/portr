package db

import (
	"time"

	"gorm.io/gorm"
)

type JsonField map[string]any

type UserRole string

const (
	SuperUser UserRole = "superuser"
	Admin     UserRole = "admin"
	Member    UserRole = "member"
)

type User struct {
	gorm.Model

	Email     string   `gorm:"uniqueIndex"`
	FirstName *string  `gorm:"null"`
	LastName  *string  `gorm:"null"`
	SecretKey *string  `gorm:"null" json:"-"`
	Role      UserRole `gorm:"default:member"`
}

type Session struct {
	gorm.Model

	Token  string
	UserID uint
	User   User
}

type OAuthState struct {
	gorm.Model

	AccessToken string
	Metadata    JsonField `gorm:"type:json"`
	UserID      uint
	User        User
}

type InviteStatus string

const (
	Invited  InviteStatus = "invited"
	Accepted InviteStatus = "accepted"
	Expired  InviteStatus = "expired"
)

type Invite struct {
	gorm.Model

	Email           string       `json:"email"`
	Status          InviteStatus `gorm:"default:invited"`
	Role            UserRole     `gorm:"default:member"`
	InvitedByUserID uint
	InvitedByUser   User
}

type Connection struct {
	gorm.Model

	Subdomain string
	ClosedAt  *time.Time
	UserID    uint
	User      User
}
