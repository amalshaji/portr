package main

import (
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/urfave/cli/v2"
)

func configCmd() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Edit the portr config file",
		Subcommands: []*cli.Command{
			{
				Name:  "edit",
				Usage: "Edit the default config file",
				Action: func(c *cli.Context) error {
					// This command opens (and may create) config files on disk. Disallow it when file backed configs are disabled.
					if c.Bool("disable-config") {
						return cli.Exit("config edit cannot be used with --disable-config", 1)
					}
					return config.EditConfig()
				},
			},
		},
	}
}
