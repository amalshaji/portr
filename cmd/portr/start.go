package main

import (
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
	config, err := config.Load(c.String("config"))
	if err != nil {
		return err
	}

	db := db.New(&config)

	_c := client.NewClient(&config, db)

	if tunnelFromCli != nil {
		tunnelFromCli.SetDefaults()
		if err := tunnelFromCli.Validate(); err != nil {
			return err
		}
		_c.ReplaceTunnelsFromCli(*tunnelFromCli)
		err = _c.Start(c.Context)
	} else {
		if err := config.Validate(); err != nil {
			return err
		}
		err = _c.Start(c.Context, c.Args().Slice()...)
	}

	if err != nil {
		return err
	}

	dash := dashboard.New(db, _c.GetConfig())
	go func() {
		if err := dash.Start(); err != nil {
			log.Fatalf("Failed to start dashboard server: error: %v", err)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh

	_c.Shutdown(c.Context)
	dash.Shutdown()
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
