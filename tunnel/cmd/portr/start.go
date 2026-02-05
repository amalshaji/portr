package main

import (
	"fmt"
	"log"

	"os"
	"os/signal"
	"syscall"

	"github.com/amalshaji/portr/internal/client/client"
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/dashboard"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/urfave/cli/v2"
)

func startTunnels(c *cli.Context, tunnelFromCli *config.Tunnel) error {
	var cfg config.Config
	var err error

	if c.Bool("disable-config") {
		// --config is not allowed when file backed configs are disabled.
		if c.IsSet("config") {
			return fmt.Errorf("--config cannot be used with --disable-config")
		}

		token := c.String("token")
		remote := c.String("remote")
		if token == "" || remote == "" {
			return fmt.Errorf("--token and --remote are required with --disable-config")
		}

		cfg, err = config.LoadFromRemote(token, remote)
		if err != nil {
			return err
		}
	} else {
		cfg, err = config.Load(c.String("config"))
		if err != nil {
			return err
		}
	}

	// CLI/env overrides for TUI settings.
	if c.IsSet("disable-tui") && c.Bool("disable-tui") {
		cfg.DisableTUI = true
	}

	// CLI overrides for local dashboard settings.
	if c.IsSet("disable-dashboard") && c.Bool("disable-dashboard") {
		cfg.DisableDashboard = true
	}

	db := db.New(&cfg)

	_c := client.NewClient(&cfg, db)

	if tunnelFromCli != nil {
		tunnelFromCli.SetDefaults()
		if err := tunnelFromCli.Validate(); err != nil {
			return err
		}
		_c.ReplaceTunnelsFromCli(*tunnelFromCli)
		err = _c.Start(c.Context)
	} else {
		if err := cfg.Validate(); err != nil {
			return err
		}
		err = _c.Start(c.Context, c.Args().Slice()...)
	}

	if err != nil {
		return err
	}

	var dash *dashboard.Dashboard

	if !_c.GetConfig().DisableDashboard {
		dash = dashboard.New(db, _c.GetConfig())
		_c.SetDashboardURL(fmt.Sprintf("http://localhost:%d", dash.Port()))
		go func() {
			if err := dash.Start(); err != nil {
				log.Fatalf("Failed to start dashboard server: error: %v", err)
			}
		}()
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh

	_c.Shutdown(c.Context)

	if dash != nil {
		dash.Shutdown()
	}

	return nil
}

func startCmd() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start the tunnels from the config file",
		Action: func(c *cli.Context) error {
			return startTunnels(c, nil)
		},
	}
}
