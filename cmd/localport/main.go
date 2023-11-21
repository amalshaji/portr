package main

import (
	"fmt"
	"os"

	"github.com/labstack/gommon/color"
	"github.com/urfave/cli/v2"
)

const VERSION = "0.0.1-beta"

func main() {
	app := &cli.App{
		Name:    "rig",
		Usage:   "Expose local http/tcp servers to the internet",
		Version: VERSION,
		Commands: []*cli.Command{
			startCmd(),
			configCmd(),
			httpCmd(),
			tcpCmd(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(color.Red(err.Error()))
		os.Exit(0)
	}
}
