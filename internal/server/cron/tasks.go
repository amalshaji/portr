package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/amalshaji/localport/internal/server/db"
)

type Job struct {
	Name     string
	Interval time.Duration
	Function func(*db.Db, *slog.Logger)
}

var crons = []Job{
	{
		Name:     "Delete expired sessions",
		Interval: 6 * time.Hour,
		Function: func(db *db.Db, logger *slog.Logger) {
			if err := db.Queries.DeleteExpiredSessions(context.Background()); err != nil {
				logger.Error("error deleting expired sessions", "error", err)
			}
		},
	},
	{
		Name:     "Delete unclaimed connections",
		Interval: 10 * time.Second,
		Function: func(db *db.Db, logger *slog.Logger) {
			if err := db.Queries.DeleteUnclaimedConnections(context.Background()); err != nil {
				logger.Error("error deleting unclaimed connections", "error", err)
			}
		},
	},
}
