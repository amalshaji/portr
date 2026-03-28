package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Session struct {
	Model
	Timestamps
	UserID    uint      `gorm:"not null" json:"user_id"`
	User      User      `json:"user,omitempty"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
}

func (Session) TableName() string {
	return "session"
}

func GenerateSessionToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func NewSession(userID uint) *Session {
	return &Session{
		UserID:    userID,
		Token:     GenerateSessionToken(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *Session) Extend() {
	s.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
}
