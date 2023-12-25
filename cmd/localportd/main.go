package main

import (
	"log"
	"os"

	"github.com/amalshaji/localport/internal/server/admin"
	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/server/proxy"
	"github.com/amalshaji/localport/internal/server/smtp"
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

	db := db.New()
	db.Connect(config)

	smtp := smtp.New(&config.Admin)

	service := service.New(db, config, smtp)

	proxyServer := proxy.New(config)
	sshServer := sshd.New(&config.Ssh, proxyServer, service)
	adminServer := admin.New(config, service)

	go proxyServer.Start()
	go sshServer.Start()
	adminServer.Start()
}
