package db

import (
	"time"
)

type Connection struct {
	ID          string `gorm:"primarykey"`
	Type        string
	Subdomain   *string
	Port        *uint32
	Status      string
	CreatedAt   time.Time
	StartedAt   *time.Time
	ClosedAt    *time.Time
	Credentials []byte
	CreatedByID uint
	CreatedBy   TeamUser
}

func (Connection) TableName() string {
	return "connection"
}

type TeamUser struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	SecretKey string
	Role      string
	TeamID    uint32
	UserID    uint32
}

func (TeamUser) TableName() string {
	return "team_users"
}
