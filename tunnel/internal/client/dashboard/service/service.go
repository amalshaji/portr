package service

import (
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
)

type Service struct {
	db     *db.Db
	config *config.Config
}

func New(db *db.Db, config *config.Config) *Service {
	return &Service{
		db:     db,
		config: config,
	}
}
