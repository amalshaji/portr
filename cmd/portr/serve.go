package main

import (
	"fmt"
	"strings"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/urfave/cli/v2"
)

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:      "serve",
		Usage:     "Serve a local directory over a public url",
		ArgsUsage: "<directory>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "subdomain",
				Aliases: []string{"s"},
				Usage:   "Subdomain to serve the directory from",
			},
			basicAuthFlag(),
		},
		Action: func(c *cli.Context) error {
			dir := strings.TrimSpace(c.Args().First())
			if dir == "" {
				return fmt.Errorf("please specify a directory to serve")
			}

			return startTunnels(c, &config.Tunnel{
				Dir:       dir,
				Subdomain: c.String("subdomain"),
				Type:      constants.Static,
				BasicAuth: c.String("basic-auth"),
			})
		},
	}
}
