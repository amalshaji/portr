package main

import (
	"fmt"
	"os"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/labstack/gommon/color"
	"github.com/urfave/cli/v2"
)

// Set at build time
var version = "0.0.0"

func main() {
	app := &cli.App{
		Name:    "portr",
		Usage:   "Expose local ports to the public internet",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Config file",
				Value:   config.DefaultConfigPath,
			},
			&cli.BoolFlag{
				Name:    "disable-config",
				Usage:   "Disable file backed configs (do not read or write config files)",
				EnvVars: []string{"PORTR_DISABLE_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "token",
				Usage:   "Auth token used to download config (used with --disable-config)",
				EnvVars: []string{"PORTR_AUTH_TOKEN"},
			},
			&cli.StringFlag{
				Name:    "remote",
				Usage:   "Remote server URL used to download config (used with --disable-config)",
				EnvVars: []string{"PORTR_AUTH_REMOTE"},
			},
			&cli.BoolFlag{
				Name:    "disable-tui",
				Usage:   "Disable the terminal UI (TUI)",
				EnvVars: []string{"PORTR_DISABLE_TUI"},
			},
			&cli.BoolFlag{
				Name:    "disable-dashboard",
				Usage:   "Disable local dashboard server",
				EnvVars: []string{"PORTR_DISABLE_DASHBOARD"},
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

	app.Before = func(c *cli.Context) error {
		// for debugging cli commands
		// because the config file is not loaded when this is set
		debugForCli := os.Getenv("DEBUG_FOR_CLI") == "1"

		// When config is disabled we must not touch disk.
		if c.Bool("disable-config") {
			// If the user explicitly set a config path, error out.
			if c.IsSet("config") {
				return fmt.Errorf("--config cannot be used with --disable-config")
			}
			return nil
		}

		if err := utils.EnsureDirExists(config.DefaultConfigDir); err != nil {
			return err
		}

		// Load config to check if update checks are disabled
		cfg, configErr := config.Load(c.String("config"))
		disableUpdateCheck := configErr == nil && cfg.DisableUpdateCheck

		if !disableUpdateCheck {
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
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(color.Red(err.Error()))
		os.Exit(0)
	}
}
