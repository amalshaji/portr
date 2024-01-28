package main

import (
	"context"
	"log"
	"os"

	"github.com/amalshaji/portr/internal/server/admin"
	"github.com/amalshaji/portr/internal/server/admin/service"
	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/cron"
	"github.com/amalshaji/portr/internal/server/db"

	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/smtp"
	sshd "github.com/amalshaji/portr/internal/server/ssh"
	"github.com/urfave/cli/v2"
)

const VERSION = "0.0.1-beta"

func main() {
	app := &cli.App{
		Name:    "portrd",
		Usage:   "portr server",
		Version: VERSION,
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start the portr server",
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
			{
				Name:  "migrate",
				Usage: "Run database migrations",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "config file",
						Value:   "config.yaml",
					},
				},
				Action: func(c *cli.Context) error {
					migrate(c.String("config"))
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

	_db := db.New(&config.Database)
	_db.Connect()
	migrator := db.NewMigrator(_db, &config.Database)

	if config.Database.AutoMigrate {
		if err := migrator.Migrate(); err != nil {
			log.Fatal(err)
		}
		_db.PopulateDefaultSettings(context.Background())
	}

	smtp := smtp.New(&config.Admin)

	service := service.New(_db, config, smtp)

	proxyServer := proxy.New(config)
	sshServer := sshd.New(&config.Ssh, proxyServer, service)
	adminServer := admin.New(config, service)
	cron := cron.New(_db, config)

	go proxyServer.Start()
	go sshServer.Start()
	go cron.Start()
	adminServer.Start()
}

func migrate(configFilePath string) {
	config, err := config.Load(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	_db := db.New(&config.Database)
	_db.Connect()
	migrator := db.NewMigrator(_db, &config.Database)
	if err := migrator.Migrate(); err != nil {
		log.Fatal(err)
	}
	_db.PopulateDefaultSettings(context.Background())
}
