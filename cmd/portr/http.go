package main

import (
	"fmt"
	"strconv"
	"strings"

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
		},
		Action: func(c *cli.Context) error {
			tunnel, err := httpTunnelFromContext(c)
			if err != nil {
				return err
			}

			return startTunnels(c, tunnel)
		},
	}
}

func httpTunnelFromContext(c *cli.Context) (*config.Tunnel, error) {
	portStr := c.Args().First()

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("please specify a valid port")
	}

	return &config.Tunnel{
		Port:       port,
		Subdomain:  httpSubdomainFromContext(c),
		Type:       constants.Http,
		HostHeader: c.String("host-header"),
	}, nil
}

func httpSubdomainFromContext(c *cli.Context) string {
	if subdomain := c.String("subdomain"); subdomain != "" {
		return subdomain
	}

	args := c.Args().Slice()
	for i, arg := range args {
		switch {
		case arg == "-s" || arg == "--subdomain":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "-s="):
			return strings.TrimPrefix(arg, "-s=")
		case strings.HasPrefix(arg, "--subdomain="):
			return strings.TrimPrefix(arg, "--subdomain=")
		}
	}

	return ""
}
