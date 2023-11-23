package service

import (
	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/server/db"
)

type Service struct {
	db     *db.Db
	config *config.Config
}

func New(db *db.Db, config *config.Config) *Service {
	return &Service{db: db, config: config}
}
