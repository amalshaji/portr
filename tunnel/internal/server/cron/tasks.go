package cron

import (
	"context"
	"time"
)

type CronFunc func(*Cron)

type Job struct {
	Name     string
	Interval time.Duration
	Function CronFunc
}

var crons = []Job{
	{
		Name:     "Ping active connections",
		Interval: 10 * time.Second,
		Function: func(c *Cron) {
			c.pingActiveConnections(context.Background())
		},
	},
}
