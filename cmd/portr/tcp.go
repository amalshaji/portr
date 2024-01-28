package main

import (
	"fmt"
	"strconv"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/urfave/cli/v2"
)

func tcpCmd() *cli.Command {
	return &cli.Command{
		Name:  "tcp",
		Usage: "Expose tcp port",
		Action: func(c *cli.Context) error {
			portStr := c.Args().First()

			port, err := strconv.Atoi(portStr)
			if err != nil {
				return fmt.Errorf("please specify a valid port")
			}

			return startTunnels(c, &config.Tunnel{
				Port:      port,
				Subdomain: "",
				Type:      constants.Tcp,
			})
		},
	}
}
