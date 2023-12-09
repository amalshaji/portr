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
	SecretKey string   `gorm:"null"`
	Role      UserRole `gorm:"default:member"`

	GithubAccessToken string `json:"-"`
	GithubAvatarUrl   string `json:"AvatarUrl"`
}

type Session struct {
	gorm.Model

	Token  string
	UserID uint
	User   User
}

type InviteStatus string

const (
	Active    InviteStatus = "active"
	Accepted  InviteStatus = "accepted"
	Cancelled InviteStatus = "cancelled"
)

type Invite struct {
	gorm.Model

	Email           string
	Role            UserRole     `gorm:"default:member"`
	Status          InviteStatus `gorm:"default:active"`
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

type Settings struct {
	gorm.Model

	SignupRequiresInvite           bool
	AllowRandomUserSignup          bool
	RandomUserSignupAllowedDomains string
	UserInviteEmailTemplate        string
}
