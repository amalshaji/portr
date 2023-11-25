package service

import (
	"log/slog"

	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/server/smtp"
	"github.com/amalshaji/localport/internal/utils"
)

type Service struct {
	db     *db.Db
	config *config.Config
	smtp   *smtp.Smtp
	log    *slog.Logger
}

func New(db *db.Db, config *config.Config, smtp *smtp.Smtp) *Service {
	return &Service{db: db, config: config, smtp: smtp, log: utils.GetLogger()}
}
