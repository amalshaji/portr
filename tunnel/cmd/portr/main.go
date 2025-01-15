package main

import (
	"fmt"
	"os"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/labstack/gommon/color"
	"github.com/urfave/cli/v2"
)

const VERSION = "0.0.27-beta"

func main() {
	app := &cli.App{
		Name:    "portr",
		Usage:   "Expose local ports to the public internet",
		Version: VERSION,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Config file",
				Value:   config.DefaultConfigPath,
			},
		},
		Commands: []*cli.Command{
			startCmd(),
			configCmd(),
			httpCmd(),
			tcpCmd(),
			authCmd(),
		},
	}

	if err := utils.EnsureDirExists(config.DefaultConfigDir); err != nil {
		fmt.Println(color.Red(err.Error()))
		os.Exit(0)
	}

	// for debugging cli commands
	// because the config file is not loaded when this is set
	debugForCli := os.Getenv("DEBUG_FOR_CLI") == "1"

	go func() {
		if err := checkForUpdates(); err != nil {
			if debugForCli {
				fmt.Println(color.Red(err.Error()))
			}
		}
	}()

	versionToUpdate, err := getVersionToUpdate()
	if err != nil {
		if debugForCli {
			fmt.Println(color.Red(err.Error()))
		}
	} else {
		if versionToUpdate != "" {
			fmt.Printf(color.Yellow("A new version of Portr is available: %s. https://github.com/amalshaji/portr/releases/tag/%s\n"), versionToUpdate, versionToUpdate)
		}
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(color.Red(err.Error()))
		os.Exit(0)
	}
}
