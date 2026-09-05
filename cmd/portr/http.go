package main

import (
	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/urfave/cli/v2"
)

func httpCmd() *cli.Command {
	return &cli.Command{
		Name:      "http",
		Usage:     "Expose http/ws port",
		ArgsUsage: "<port | host:port>",
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
			basicAuthFlag(),
		},
		Action: func(c *cli.Context) error {
			host, port, err := parseLocalTarget(c.Args().First())
			if err != nil {
				return err
			}

			return startTunnels(c, &config.Tunnel{
				Host:       host,
				Port:       port,
				Subdomain:  c.String("subdomain"),
				Type:       constants.Http,
				HostHeader: c.String("host-header"),
				BasicAuth:  c.String("basic-auth"),
			})
		},
	}
}
