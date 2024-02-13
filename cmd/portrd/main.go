package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

type ServiceToRun int

const (
	ADMIN_SERVER  ServiceToRun = iota + 1
	TUNNEL_SERVER              = iota + 1
	ALL_SERVERS                = iota + 1
)

func main() {
	app := &cli.App{
		Name:    "portrd",
		Usage:   "portr server",
		Version: VERSION,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config file",
				Value:   "config.yaml",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Specify the server to start",
				Subcommands: []*cli.Command{
					{
						Name:  "admin",
						Usage: "Start the admin server",
						Action: func(c *cli.Context) error {
							start(c.String("config"), ADMIN_SERVER)
							return nil
						},
					},
					{
						Name:  "tunnel",
						Usage: "Start the tunnel server",
						Action: func(c *cli.Context) error {
							start(c.String("config"), TUNNEL_SERVER)
							return nil
						},
					},
					{
						Name:  "all",
						Usage: "Start both admin and tunnel servers",
						Action: func(c *cli.Context) error {
							start(c.String("config"), ALL_SERVERS)
							return nil
						},
					},
				},
			},
			{
				Name:  "migrate",
				Usage: "Run database migrations",
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

func start(configFilePath string, toRun ServiceToRun) {
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

	if toRun == TUNNEL_SERVER || toRun == ALL_SERVERS {
		go proxyServer.Start()
		defer proxyServer.Shutdown(context.TODO())

		go sshServer.Start()
		defer sshServer.Shutdown(context.TODO())
	}

	if toRun == ADMIN_SERVER || toRun == ALL_SERVERS {
		go adminServer.Start()
		defer adminServer.Shutdown()

		go cron.Start()
		defer cron.Shutdown()
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
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
