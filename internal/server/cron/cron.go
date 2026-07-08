package cron

import (
	"context"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
)

type Cron struct {
	config     *config.Config
	service    *service.Service
	cancelFunc context.CancelFunc
	httpClient *resty.Client
}

func New(config *config.Config, service *service.Service) *Cron {
	return &Cron{
		config:     config,
		service:    service,
		httpClient: resty.New().SetTimeout(5 * time.Second),
	}
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
					job.Function(ctx, c)
				case <-ctx.Done():
					ticker.Stop()
					return
				}
			}
		}(job)
	}
}

func (c *Cron) Shutdown() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}
