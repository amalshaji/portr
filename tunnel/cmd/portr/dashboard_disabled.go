//go:build nodashboard

package main

import (
	"github.com/amalshaji/portr/internal/client/client"
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/urfave/cli/v2"
)

func dashboardFlags() []cli.Flag {
	return nil
}

func applyDashboardCLIOverrides(_ *cli.Context, _ *config.Config) {
}

func startDashboardIfEnabled(_ *client.Client, _ *db.Db) (shutdown func(), err error) {
	return nil, nil
}
