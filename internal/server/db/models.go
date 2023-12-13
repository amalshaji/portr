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

	Email     string  `gorm:"uniqueIndex"`
	FirstName *string `gorm:"null"`
	LastName  *string `gorm:"null"`

	IsSuperUser bool `gorm:"default:false"`

	GithubAccessToken string `json:"-"`
	GithubAvatarUrl   string `json:"AvatarUrl"`

	Teams []Team `gorm:"many2many:team_users"`
}

type Team struct {
	gorm.Model

	Name string `gorm:"uniqueIndex"`
	Slug string `gorm:"uniqueIndex"`

	Users []User `gorm:"many2many:team_users"`
}

type TeamUser struct {
	gorm.Model

	TeamID uint
	Team   Team
	UserID uint
	User   User

	SecretKey string   `gorm:"uniqueIndex"`
	Role      UserRole `gorm:"default:member"`
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

	Email               string
	Role                UserRole     `gorm:"default:member"`
	Status              InviteStatus `gorm:"default:active"`
	InvitedByTeamUserID uint
	InvitedByTeamUser   TeamUser

	TeamID uint
	Team   Team
}

type Connection struct {
	gorm.Model

	Subdomain  string
	ClosedAt   *time.Time
	TeamUserID uint
	TeamUser   TeamUser
}

type Settings struct {
	gorm.Model

	UserInviteEmailTemplate string

	TeamID uint
	Team   Team
}
