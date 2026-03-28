package main

import (
	"fmt"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/labstack/gommon/color"

	"github.com/urfave/cli/v2"
)

func authCmd() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Setup portr cli auth",
		Subcommands: []*cli.Command{
			{
				Name:  "set",
				Usage: "Set the cli auth token",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "token",
						Aliases:  []string{"t"},
						Usage:    "The auth token",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "remote",
						Aliases:  []string{"r"},
						Usage:    "The remote server url",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					err := config.GetConfig(c.String("token"), c.String("remote"))
					if err != nil {
						return err
					}

					fmt.Println(color.Green("Cli auth success!"))
					return nil
				},
			},
		},
	}
}
