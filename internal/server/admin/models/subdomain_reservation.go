package models

import "time"

type SubdomainReservation struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Subdomain  string    `gorm:"not null" json:"subdomain"`
	TeamUserID uint      `gorm:"not null;index" json:"team_user_id"`
	TeamUser   TeamUser  `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
}

func (SubdomainReservation) TableName() string {
	return "subdomain_reservation"
}
