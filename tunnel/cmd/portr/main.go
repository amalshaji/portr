package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/amalshaji/portr/internal/client/config"
	requestlogs "github.com/amalshaji/portr/internal/client/logs"
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
		},
		Commands: []*cli.Command{
			startCmd(),
			configCmd(),
			httpCmd(),
			tcpCmd(),
			logsCmd(),
			authCmd(),
		},
	}

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
	suppressUpdateNotice := shouldSuppressUpdateNotice(os.Args)

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

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, color.Red(err.Error()))
		os.Exit(1)
	}
}

func shouldSuppressUpdateNotice(args []string) bool {
	commandArgs, ok := commandArgs(args, "logs")
	if !ok {
		return false
	}

	if requestlogs.WantsHelp(commandArgs) {
		return true
	}

	return requestlogs.WantsJSON(commandArgs)
}

func commandArgs(args []string, command string) ([]string, bool) {
	for i := 1; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}

		switch {
		case arg == "--config" || arg == "-c":
			i++
			continue
		case strings.HasPrefix(arg, "--config="):
			continue
		case strings.HasPrefix(arg, "-"):
			continue
		}

		if arg == command {
			return args[i+1:], true
		}

		return nil, false
	}

	return nil, false
}
