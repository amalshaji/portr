package service

import (
	"fmt"

	"github.com/amalshaji/localport/internal/server/db"
)

func (s *Service) ValidateClientConfig(key string) error {
	var count int64
	result := s.db.Conn.Find(&db.User{}, "secret_key = ?", key).Count(&count)
	if result.Error != nil {
		s.log.Error("error while validating secret key", "error", result.Error)
		return result.Error
	}
	if count == 0 {
		s.log.Error("invalid secret key", "key", key)
		return fmt.Errorf("invalid secret key")
	}
	return nil
}
