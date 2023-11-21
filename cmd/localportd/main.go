package main

import (
	"log"
	"os"

	"github.com/amalshaji/localport/internal/server/admin"
	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/server/proxy"
	sshd "github.com/amalshaji/localport/internal/server/ssh"
	"github.com/urfave/cli/v2"
)

const VERSION = "0.0.1-beta"

func main() {
	app := &cli.App{
		Name:    "localportd",
		Usage:   "Expose local http/ws servers to the internet",
		Version: VERSION,
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start the localport server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "config file",
						Value:   "config.yaml",
					},
				},
				Action: func(c *cli.Context) error {
					start(c.String("config"))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func start(configFilePath string) {
	config, err := config.Load(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	proxyServer := proxy.New(config)
	sshServer := sshd.New(&config.Ssh, proxyServer)
	adminServer := admin.New(&config.Admin)

	go proxyServer.Start()
	go sshServer.Start()
	adminServer.Start()
}
