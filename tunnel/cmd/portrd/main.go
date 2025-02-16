package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/cron"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/service"
	sshd "github.com/amalshaji/portr/internal/server/ssh"
	"github.com/urfave/cli/v2"
)

const VERSION = "0.0.28-beta"

func main() {
	app := &cli.App{
		Name:    "portrd",
		Usage:   "portr server",
		Version: VERSION,
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start the tunnel server",
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
	config := config.Load(configFilePath)

	_db := db.New(&config.Database)
	_db.Connect()

	service := service.New(_db)

	proxyServer := proxy.New(config)
	sshServer := sshd.New(&config.Ssh, proxyServer, service)
	cron := cron.New(_db, config, service)

	go proxyServer.Start()
	defer proxyServer.Shutdown(context.TODO())

	go sshServer.Start()
	defer sshServer.Shutdown(context.TODO())

	go cron.Start()
	defer cron.Shutdown()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
}
