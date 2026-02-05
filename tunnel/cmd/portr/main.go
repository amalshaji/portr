package main

import (
	"fmt"
	"os"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
)

// Set at build time
var version = "0.0.0"
var appName = "portr"

func main() {
	app := &cli.App{
		Name:    appName,
		Usage:   "Expose local ports to the public internet",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Config file",
				Value:   config.DefaultConfigPath,
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				EnvVars: []string{"LOG_LEVEL"},
			},
			&cli.BoolFlag{
				Name:    "disable-config",
				Usage:   "Disable file backed configs (do not read or write config files)",
				EnvVars: []string{"PORTR_DISABLE_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "token",
				Usage:   "Auth token used to download config (used with --disable-config)",
				EnvVars: []string{"PORTR_TOKEN"},
			},
			&cli.StringFlag{
				Name:    "remote",
				Usage:   "Remote server URL used to download config (used with --disable-config)",
				EnvVars: []string{"PORTR_REMOTE"},
			},
			&cli.BoolFlag{
				Name:    "disable-tui",
				Usage:   "Disable the terminal UI (TUI)",
				EnvVars: []string{"PORTR_DISABLE_TUI"},
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
	app.Flags = append(app.Flags, dashboardFlags()...)

	app.Before = func(c *cli.Context) error {
		// Configure logger early so startup errors are consistent.
		log.SetOutput(os.Stdout)
		log.SetFormatter(log.TextFormatter)
		log.SetReportTimestamp(true)
		log.SetTimeFormat(log.DefaultTimeFormat)
		log.SetReportCaller(false)
		log.SetCallerOffset(1)

		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			return fmt.Errorf("invalid --log-level: %w", err)
		}
		log.SetLevel(level)

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
						log.Debug("Update check failed", "error", err)
					}
				}
			}()

			versionToUpdate, err := getVersionToUpdate()
			if err != nil {
				if debugForCli {
					log.Debug("Failed to determine update version", "error", err)
				}
			} else {
				if versionToUpdate != "" {
					log.Info("A new version of Portr is available", "version", versionToUpdate, "release", "https://github.com/amalshaji/portr/releases/tag/"+versionToUpdate)
				}
			}
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error("Command failed", "error", err)
		os.Exit(0)
	}
}
