package main

import (
	"fmt"
	"strconv"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
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
			&cli.StringFlag{
				Name:  "host-header",
				Usage: "Host header to send to the local server ('rewrite' to use the local address)",
			},
			&cli.StringFlag{
				Name:    "basic-auth",
				Usage:   "Protect the tunnel with HTTP basic auth, as user:password",
				EnvVars: []string{"PORTR_BASIC_AUTH"},
			},
		},
		Action: func(c *cli.Context) error {
			portStr := c.Args().First()

			port, err := strconv.Atoi(portStr)
			if err != nil {
				return fmt.Errorf("please specify a valid port")
			}

			return startTunnels(c, &config.Tunnel{
				Port:       port,
				Subdomain:  c.String("subdomain"),
				Type:       constants.Http,
				HostHeader: c.String("host-header"),
				BasicAuth:  c.String("basic-auth"),
			})
		},
	}
}
