package main

import (
	"fmt"
	"os"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/labstack/gommon/color"
	"github.com/urfave/cli/v2"
)

const VERSION = "0.0.20-beta"

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

	if err := app.Run(os.Args); err != nil {
		fmt.Println(color.Red(err.Error()))
		os.Exit(0)
	}
}
