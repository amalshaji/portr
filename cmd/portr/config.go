package main

import (
	"fmt"
	"strings"

	config "github.com/amalshaji/portr/internal/clientconfig"
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
				Name:  "pull",
				Usage: "Replace the local tunnels and groups with the team template",
				Action: func(c *cli.Context) error {
					changes, err := config.PullConfig()
					if err != nil {
						return err
					}

					printTemplateChanges(changes)
					return nil
				},
			},
		},
	}
}

func printTemplateChanges(changes config.TemplateChanges) {
	fmt.Println(color.Green("Pulled the team client template into " + config.DefaultConfigPath))
	fmt.Println("tunnels: " + strings.Join(changes.Tunnels, ", "))
	if len(changes.Added) > 0 {
		fmt.Println("  added:   " + color.Green(strings.Join(changes.Added, ", ")))
	}
	if len(changes.Removed) > 0 {
		fmt.Println("  removed: " + color.Yellow(strings.Join(changes.Removed, ", ")))
	}
	if len(changes.Groups) > 0 {
		fmt.Println("groups:  " + strings.Join(changes.Groups, ", "))
	}
}
