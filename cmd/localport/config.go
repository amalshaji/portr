package main

import (
	"github.com/amalshaji/localport/internal/client/config"
	"github.com/urfave/cli/v2"
)

func configCmd() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Edit the localport config file",
		Subcommands: []*cli.Command{
			{
				Name:  "edit",
				Usage: "Edit the default config file",
				Action: func(c *cli.Context) error {
					return config.EditConfig()
				},
			},
		},
	}
}
