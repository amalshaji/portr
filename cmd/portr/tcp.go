package main

import (
	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/urfave/cli/v2"
)

func tcpCmd() *cli.Command {
	return &cli.Command{
		Name:      "tcp",
		Usage:     "Expose tcp port",
		ArgsUsage: "<port | host:port>",
		Action: func(c *cli.Context) error {
			host, port, err := parseLocalTarget(c.Args().First())
			if err != nil {
				return err
			}

			return startTunnels(c, &config.Tunnel{
				Host:      host,
				Port:      port,
				Subdomain: "",
				Type:      constants.Tcp,
			})
		},
	}
}
