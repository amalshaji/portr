package service

import (
	"log/slog"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/utils"
)

type Service struct {
	db     *db.Db
	config *config.Config
	log    *slog.Logger
}

func New(db *db.Db, config *config.Config) *Service {
	return &Service{
		db:     db,
		config: config,
		log:    utils.GetLogger(),
	}
}
