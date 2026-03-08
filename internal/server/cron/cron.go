package cron

import (
	"context"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/charmbracelet/log"
)

type Cron struct {
	db         *db.Db
	config     *config.Config
	service    *service.Service
	cancelFunc context.CancelFunc
}

func New(db *db.Db, config *config.Config, service *service.Service) *Cron {
	return &Cron{db: db, config: config, service: service}
}

func (c *Cron) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	log.Info("Starting cron jobs", "count", len(crons))
	for _, job := range crons {
		ticker := time.NewTicker(job.Interval)
		go func(job Job) {
			for {
				select {
				case <-ticker.C:
					log.Debug("Running cron job", "name", job.Name)
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
