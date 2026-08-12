package main

import (
	"fmt"
	"os"

	requestlogs "github.com/amalshaji/portr/internal/client/logs"
	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/labstack/gommon/color"
	"github.com/urfave/cli/v2"
)

// Set at build time
var version = "0.0.0"

func buildApp() *cli.App {
	return &cli.App{
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
		},
		Commands: []*cli.Command{
			startCmd(),
			configCmd(),
			httpCmd(),
			stubCmd(),
			serveCmd(),
			tcpCmd(),
			logsCmd(),
			replayCmd(),
			authCmd(),
			appServerCmd(),
			adminCmd(),
			doctorCmd(),
		},
	}
}

func main() {
	app := buildApp()

	if err := utils.EnsureDirExists(config.DefaultConfigDir); err != nil {
		fmt.Fprintln(os.Stderr, color.Red(err.Error()))
		os.Exit(1)
	}

	// for debugging cli commands
	// because the config file is not loaded when this is set
	debugForCli := os.Getenv("DEBUG_FOR_CLI") == "1"

	// Load config to check if update checks are disabled
	cfg, configErr := config.Load(config.DefaultConfigPath)
	disableUpdateCheck := configErr == nil && cfg.DisableUpdateCheck
	suppressUpdateNotice := shouldSuppressUpdateNotice(app, os.Args)

	if !disableUpdateCheck {
		go func() {
			defer func() {
				if r := recover(); r != nil && debugForCli {
					fmt.Fprintln(os.Stderr, color.Red(fmt.Sprintf("update check panic: %v", r)))
				}
			}()

			if err := checkForUpdates(); err != nil {
				if debugForCli {
					fmt.Fprintln(os.Stderr, color.Red(err.Error()))
				}
			}
		}()

		versionToUpdate, err := getVersionToUpdate()
		if err != nil {
			if debugForCli {
				fmt.Fprintln(os.Stderr, color.Red(err.Error()))
			}
		} else {
			if versionToUpdate != "" && !suppressUpdateNotice {
				fmt.Fprintf(os.Stderr, color.Yellow("A new version of Portr is available: %s. https://github.com/amalshaji/portr/releases/tag/%s\n"), versionToUpdate, versionToUpdate)
			}
		}
	}

	if err := app.Run(reorderArgs(app, os.Args)); err != nil {
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, color.Red(err.Error()))
		}
		os.Exit(1)
	}
}

func shouldSuppressUpdateNotice(app *cli.App, args []string) bool {
	idx := commandIndex(app, args)
	if idx == -1 {
		return false
	}

	rest := args[idx+1:]
	switch args[idx] {
	case "logs":
		return requestlogs.WantsHelp(rest) || requestlogs.WantsJSON(rest)
	case "replay":
		return commandWantsHelp(rest) || replayWantsJSON(rest)
	}

	return false
}
