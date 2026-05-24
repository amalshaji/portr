package main

import (
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/urfave/cli/v2"
)

func stubCmd() *cli.Command {
	return &cli.Command{
		Name:  "stub",
		Usage: "Serve a stubbed templated response through a Portr HTTP tunnel",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "subdomain",
				Aliases:  []string{"s"},
				Usage:    "Subdomain to serve the stub response from",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "response-format",
				Usage:    "Response Content-Type, for example application/json",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "response-tmpl",
				Usage: "Inline response template",
			},
			&cli.StringFlag{
				Name:  "response-tmpl-file",
				Usage: "Path to a response template file",
			},
		},
		Action: func(c *cli.Context) error {
			return startTunnels(c, &config.Tunnel{
				Subdomain:            c.String("subdomain"),
				Type:                 constants.Stub,
				ResponseFormat:       c.String("response-format"),
				ResponseTemplate:     c.String("response-tmpl"),
				ResponseTemplateFile: c.String("response-tmpl-file"),
			})
		},
	}
}
