package service

import (
	"fmt"

	"github.com/amalshaji/localport/internal/server/db"
	"gorm.io/gorm"
)

func (s *Service) ListUsers() []db.User {
	var users []db.User
	s.db.Conn.Find(&users)
	return users
}

func (s *Service) GetUserBySession(token string) (db.User, error) {
	var session = db.Session{Token: token}
	result := s.db.Conn.Joins("User").First(&session)
	if result.Error == gorm.ErrRecordNotFound {
		return db.User{}, fmt.Errorf("session not found")
	}
	return session.User, nil
}
