package models

import (
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
}

type Timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SoftDelete struct {
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
