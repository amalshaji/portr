package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amalshaji/portr/internal/client/appserver"
	"github.com/amalshaji/portr/internal/client/dashboard"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/dockerwatch"
	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
)

func dockerCmd() *cli.Command {
	return &cli.Command{
		Name:  "docker",
		Usage: "Start tunnels for compose services labelled with portr.enable=true",
		Description: "Run inside a container in the same compose project, with the docker socket mounted.\n" +
			"Tunnels follow containers: they start when a labelled service starts and stop when it dies.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "docker-host",
				Usage:   "Docker daemon address",
				Value:   dockerwatch.DefaultDockerHost,
				EnvVars: []string{"DOCKER_HOST"},
			},
			&cli.StringFlag{
				Name:    "project",
				Usage:   "Compose project to watch (default: the project of the portr container)",
				EnvVars: []string{"PORTR_DOCKER_PROJECT"},
			},
		},
		Action: startDocker,
	}
}

func dockerConfig(path string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, err
	}

	// Tunnels come from labels only, so a shared config's own tunnels must not
	// fail validation inside the container (static dirs, for one).
	if len(cfg.Tunnels) > 0 {
		log.Info("Ignoring tunnels from the config file; docker mode reads them from labels", "count", len(cfg.Tunnels))
	}
	cfg.Tunnels = nil
	cfg.Groups = nil
	cfg.DisableTUI = true

	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func startDocker(c *cli.Context) error {
	cfg, err := dockerConfig(c.String("config"))
	if err != nil {
		return err
	}

	api, err := dockerwatch.NewAPI(c.String("docker-host"))
	if err != nil {
		return err
	}

	database := db.New(&cfg)
	manager := appserver.NewManager(cfg, database)
	manager.SetEventLogger(log.WithPrefix("docker"), "Tunnel event")
	watcher := dockerwatch.New(api, manager, dockerwatch.Options{Project: c.String("project")})

	ctx, cancel := signal.NotifyContext(c.Context, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var dash *dashboard.Dashboard
	dashErrCh := make(chan error, 1)
	if !cfg.DisableDashboard {
		dash = dashboard.New(database, &cfg)
		go func() {
			if err := dash.Start(); err != nil {
				dashErrCh <- fmt.Errorf("failed to start dashboard server: %w", err)
			}
		}()
	}

	watchErrCh := make(chan error, 1)
	go func() {
		watchErrCh <- watcher.Run(ctx)
	}()

	var runErr error
	select {
	case runErr = <-watchErrCh:
	case runErr = <-dashErrCh:
		cancel()
		<-watchErrCh
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	manager.Shutdown(shutdownCtx)
	if dash != nil {
		if err := dash.Shutdown(); err != nil && runErr == nil {
			runErr = err
		}
	}
	return runErr
}
