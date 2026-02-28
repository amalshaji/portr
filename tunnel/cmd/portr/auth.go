package main

import (
	"fmt"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/charmbracelet/log"

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
				Action: func(c *cli.Context) error {
					// This command writes a downloaded config to disk. Disallow it when file backed configs are disabled.
					if c.Bool("disable-config") {
						return fmt.Errorf("auth set cannot be used with --disable-config")
					}

					err := config.GetConfig(c.String("token"), c.String("remote"))
					if err != nil {
						return err
					}

					log.Info("CLI auth success")
					return nil
				},
			},
		},
	}
}
