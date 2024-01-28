package main

import (
	"fmt"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/labstack/gommon/color"
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
					err = config.ValidateConfig()
					if err != nil {
						return err
					}

					fmt.Println(color.Green("Config file is valid"))
					return nil
				},
			},
		},
	}
}
