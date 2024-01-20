package main

import (
	"fmt"
	"strconv"

	"github.com/amalshaji/localport/internal/client/config"
	"github.com/amalshaji/localport/internal/constants"
	"github.com/urfave/cli/v2"
)

func httpCmd() *cli.Command {
	return &cli.Command{
		Name:  "http",
		Usage: "Expose http/ws port",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "subdomain",
				Aliases: []string{"s"},
				Usage:   "Subdomain to tunnel to",
			},
		},
		Action: func(c *cli.Context) error {
			portStr := c.Args().First()

			port, err := strconv.Atoi(portStr)
			if err != nil {
				return fmt.Errorf("please specify a valid port")
			}

			return startTunnels(c, &config.Tunnel{
				Port:      port,
				Subdomain: c.String("subdomain"),
				Type:      constants.Http,
			})
		},
	}
}
