package cron

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/amalshaji/portr/internal/utils"
)

type Cron struct {
	db         *db.Db
	logger     *slog.Logger
	config     *config.Config
	service    *service.Service
	cancelFunc context.CancelFunc
}

func New(db *db.Db, config *config.Config, service *service.Service) *Cron {
	return &Cron{db: db, config: config, service: service, logger: utils.GetLogger()}
}

func (c *Cron) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	c.logger.Info(fmt.Sprintf("Starting %d cron jobs", len(crons)))
	for _, job := range crons {
		ticker := time.NewTicker(job.Interval)
		go func(job Job) {
			for {
				select {
				case <-ticker.C:
					c.logger.Debug("Running cron job: " + job.Name)
					job.Function(c)
				case <-ctx.Done():
					ticker.Stop()
					return
				}
			}
		}(job)
	}
}

func (c *Cron) Shutdown() {
	c.cancelFunc()
}
