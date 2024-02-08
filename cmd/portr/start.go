package main

import (
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
	db := db.New()

	_c := client.NewClient(c.String("config"), db)

	tunnelFromCli.SetDefaults()

	dashboard := dashboard.New(db, _c.GetConfig())
	go dashboard.Start()

	if tunnelFromCli != nil {
		_c.ReplaceTunnelsFromCli(*tunnelFromCli)
		_c.Start(c.Context)
	} else {
		_c.Start(c.Context, c.Args().Slice()...)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh

	_c.Shutdown(c.Context)
	dashboard.Shutdown()
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
