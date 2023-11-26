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
			{
				Name:  "validate",
				Usage: "Validate the config file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Config file",
						Value:   config.DefaultConfigPath,
					},
				},
				Action: func(c *cli.Context) error {
					config, err := config.Load(c.String("config"))
					if err != nil {
						return err
					}
					return config.ValidateConfig()
				},
			},
		},
	}
}
