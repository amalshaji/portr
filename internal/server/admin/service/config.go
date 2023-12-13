package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Service) ValidateClientConfig(ctx context.Context, key string) error {
	_, err := s.db.Queries.GetTeamUserBySecretKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Error("invalid secret key", "key", key)
			return fmt.Errorf("invalid secret key")
		}
		return fmt.Errorf("error while validating secret key: %w", err)
	}
	return nil
}
