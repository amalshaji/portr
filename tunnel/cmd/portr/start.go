package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/amalshaji/portr/internal/client/client"
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/urfave/cli/v2"
)

func startTunnels(c *cli.Context, tunnelFromCli *config.Tunnel) error {
	_c := client.NewClient(c.String("config"))

	var err error

	if tunnelFromCli != nil {
		tunnelFromCli.SetDefaults()
		_c.ReplaceTunnelsFromCli(*tunnelFromCli)
		err = _c.Start(c.Context)
	} else {
		err = _c.Start(c.Context, c.Args().Slice()...)
	}

	if err != nil {
		return err
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh

	_c.Shutdown(c.Context)
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
