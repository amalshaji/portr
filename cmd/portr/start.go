package main

import (
	"context"
	"fmt"
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
	cfg, err := config.Load(c.String("config"))
	if err != nil {
		return err
	}

	if tunnelFromCli != nil {
		tunnelFromCli.SetDefaults()
		if err := tunnelFromCli.Validate(); err != nil {
			return err
		}
		cfg.Tunnels = []config.Tunnel{*tunnelFromCli}
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	db := db.New(&cfg)
	_c := client.NewClient(&cfg, db)
	var dash *dashboard.Dashboard
	var dashErrCh <-chan error

	defer func() {
		if r := recover(); r != nil {
			_c.Shutdown(context.Background())
			if dash != nil {
				_ = dash.Shutdown()
			}
			panic(r)
		}
	}()

	if tunnelFromCli != nil {
		err = _c.Start(c.Context)
	} else {
		err = _c.Start(c.Context, c.Args().Slice()...)
	}

	if err != nil {
		_c.Shutdown(c.Context)
		return err
	}

	if !cfg.DisableDashboard {
		dash = dashboard.New(db, _c.GetConfig())
		startErrCh := make(chan error, 1)
		dashErrCh = startErrCh
		go func() {
			defer func() {
				if r := recover(); r != nil {
					select {
					case startErrCh <- fmt.Errorf("dashboard panic: %v", r):
					default:
					}
				}
			}()
			if err := dash.Start(); err != nil {
				select {
				case startErrCh <- fmt.Errorf("failed to start dashboard server: %w", err):
				default:
				}
			}
		}()
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	var runErr error
	select {
	case <-signalCh:
	case runErr = <-_c.Done():
	case runErr = <-dashErrCh:
	}

	_c.Shutdown(c.Context)
	if dash != nil {
		if err := dash.Shutdown(); err != nil && runErr == nil {
			runErr = err
		}
	}

	return runErr
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
