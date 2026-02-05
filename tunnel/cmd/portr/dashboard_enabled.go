//go:build !nodashboard

package main

import (
	"fmt"

	"github.com/amalshaji/portr/internal/client/client"
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/dashboard"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
)

func dashboardFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "disable-dashboard",
			Usage:   "Disable local dashboard server",
			EnvVars: []string{"PORTR_DISABLE_DASHBOARD"},
		},
	}
}

func applyDashboardCLIOverrides(c *cli.Context, cfg *config.Config) {
	if c.IsSet("disable-dashboard") && c.Bool("disable-dashboard") {
		cfg.DisableDashboard = true
	}
}

func startDashboardIfEnabled(cl *client.Client, dbConn *db.Db) (shutdown func(), err error) {
	if cl.GetConfig().DisableDashboard {
		return nil, nil
	}

	dash := dashboard.New(dbConn, cl.GetConfig())
	// Use 127.0.0.1 to avoid IPv6/localhost ambiguity on some systems.
	cl.SetDashboardURL(fmt.Sprintf("http://127.0.0.1:%d", dash.Port()))

	go func() {
		if err := dash.Start(); err != nil {
			log.Error("Failed to start dashboard server", "error", err)
		}
	}()

	return dash.Shutdown, nil
}
