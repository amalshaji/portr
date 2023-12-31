package cron

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/utils"
)

type Cron struct {
	db     *db.Db
	logger *slog.Logger
}

func New(db *db.Db) *Cron {
	return &Cron{db: db, logger: utils.GetLogger()}
}

func (c *Cron) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	c.logger.Info(fmt.Sprintf("Starting %d cron jobs", len(crons)))
	for _, job := range crons {
		ticker := time.NewTicker(job.Interval)
		go func(job Job) {
			for {
				select {
				case <-ticker.C:
					c.logger.Debug("Running cron job: " + job.Name)
					job.Function(c.db, c.logger)
				case <-ctx.Done():
					ticker.Stop()
					return
				}
			}
		}(job)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	<-sigCh

	cancel()
}
