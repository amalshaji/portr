package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/amalshaji/portr/internal/client/config"
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
		},
		Action: func(c *cli.Context) error {
			portStr := c.Args().First()

			port, err := strconv.Atoi(portStr)
			if err != nil {
				return fmt.Errorf("please specify a valid port")
			}

			var subdomain, auth string

			// Urfave cli resets flags after parsing args
			// We have to do this manually
			indexOfSubdomain := slices.Index(c.Args().Slice(), "--subdomain")
			if indexOfSubdomain != -1 {
				subdomain = c.Args().Get(indexOfSubdomain + 1)
			}

			indexOfAuth := slices.Index(c.Args().Slice(), "--auth")
			if indexOfAuth != -1 {
				auth = c.Args().Get(indexOfAuth + 1)
				if auth != "" && len(strings.Split(auth, ":")) != 2 {
					return fmt.Errorf("auth must be in the format username:password")
				}
			}

			return startTunnels(c, &config.Tunnel{
				Port:      port,
				Subdomain: subdomain,
				Type:      constants.Http,
				Auth:      auth,
			})
		},
	}
}
