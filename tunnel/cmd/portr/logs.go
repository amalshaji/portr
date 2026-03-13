package main

import (
	requestlogs "github.com/amalshaji/portr/internal/client/logs"
	"github.com/urfave/cli/v2"
)

func logsCmd() *cli.Command {
	return &cli.Command{
		Name:            "logs",
		Usage:           "Read local request logs for a subdomain",
		ArgsUsage:       "<subdomain> [filter]",
		SkipFlagParsing: true,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "count",
				Aliases: []string{"n"},
				Usage:   "Number of logs to return",
				Value:   requestlogs.DefaultCount,
			},
			&cli.StringFlag{
				Name:  "since",
				Usage: "Only include logs on or after the given RFC3339 timestamp or YYYY-MM-DD date",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: requestlogs.JSONFlagUsage,
			},
		},
		Action: func(c *cli.Context) error {
			rawArgs := c.Args().Slice()
			if requestlogs.WantsHelp(rawArgs) {
				showCurrentCommandHelp(c)
				return nil
			}

			opts, err := requestlogs.ParseCommandArgs(rawArgs)
			if err != nil {
				return err
			}

			store, err := requestlogs.Open("")
			if err != nil {
				return err
			}
			defer store.Close()

			requests, err := store.List(opts.Subdomain, opts.Query)
			if err != nil {
				return err
			}

			if opts.JSON {
				return requestlogs.RenderJSON(c.App.Writer, requests)
			}

			return requestlogs.RenderText(c.App.Writer, requests)
		},
	}
}

func showCurrentCommandHelp(c *cli.Context) {
	template := cli.CommandHelpTemplate
	if len(c.Command.Subcommands) > 0 {
		template = cli.SubcommandHelpTemplate
	}
	if c.Command.CustomHelpTemplate != "" {
		template = c.Command.CustomHelpTemplate
	}

	cli.HelpPrinter(c.App.Writer, template, c.Command)
}
