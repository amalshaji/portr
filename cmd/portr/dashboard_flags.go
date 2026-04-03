package main

import (
	"fmt"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/urfave/cli/v2"
)

type dashboardOptions struct {
	PortSet     bool
	Port        int
	DisabledSet bool
	Disabled    bool
}

func dashboardFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:  "dashboard-port",
			Usage: fmt.Sprintf("Port for the local dashboard (default: %d)", config.DefaultDashboardPort),
		},
		&cli.BoolFlag{
			Name:  "disable-dashboard",
			Usage: "Disable the local dashboard server for this run",
		},
	}
}

func dashboardOptionsFromCLI(c *cli.Context) dashboardOptions {
	return dashboardOptions{
		PortSet:     c.IsSet("dashboard-port"),
		Port:        c.Int("dashboard-port"),
		DisabledSet: c.IsSet("disable-dashboard"),
		Disabled:    c.Bool("disable-dashboard"),
	}
}

func applyDashboardOptions(cfg *config.Config, opts dashboardOptions) error {
	if cfg == nil {
		return nil
	}

	if opts.PortSet && opts.DisabledSet && opts.Disabled {
		return fmt.Errorf("cannot use --dashboard-port with --disable-dashboard")
	}

	if opts.DisabledSet {
		cfg.DisableDashboard = opts.Disabled
		if opts.Disabled {
			cfg.DashboardDisableSource = config.DashboardDisableSourceCLI
		} else {
			cfg.DashboardDisableSource = ""
		}
	}

	if opts.PortSet {
		if opts.Port < 1 || opts.Port > 65535 {
			return fmt.Errorf("--dashboard-port must be between 1 and 65535")
		}

		cfg.DashboardPort = opts.Port
	}

	return nil
}
