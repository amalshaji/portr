package cron

import (
	"context"
	"time"
)

type CronFunc func(context.Context, *Cron)

type Job struct {
	Name     string
	Interval time.Duration
	Function CronFunc
}

var crons = []Job{
	{
		Name:     "Ping active connections",
		Interval: 10 * time.Second,
		Function: func(ctx context.Context, c *Cron) {
			c.pingActiveConnections(ctx)
		},
	},
}
