package service

import (
	"github.com/amalshaji/portr/internal/client/db"
	config "github.com/amalshaji/portr/internal/clientconfig"
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
