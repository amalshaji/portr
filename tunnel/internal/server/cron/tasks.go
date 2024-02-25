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
	// {
	// 	Name:     "Delete expired sessions",
	// 	Interval: 6 * time.Hour,
	// 	Function: func(c *Cron) {
	// 		if err := c.db.Queries.DeleteExpiredSessions(context.Background()); err != nil {
	// 			c.logger.Error("error deleting expired sessions", "error", err)
	// 		}
	// 	},
	// },
	// {
	// 	Name:     "Delete unclaimed connections",
	// 	Interval: 10 * time.Second,
	// 	Function: func(c *Cron) {
	// 		if err := c.db.Queries.DeleteUnclaimedConnections(context.Background()); err != nil {
	// 			c.logger.Error("error deleting unclaimed connections", "error", err)
	// 		}
	// 	},
	// },
	{
		Name:     "Ping active connections",
		Interval: 10 * time.Second,
		Function: func(c *Cron) {
			c.pingActiveConnections(context.Background())
		},
	},
}
