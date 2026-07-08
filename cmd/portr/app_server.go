package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/amalshaji/portr/internal/client/appserver"
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/urfave/cli/v2"
)

func appServerCmd() *cli.Command {
	return &cli.Command{
		Name:  "app-server",
		Usage: "Start a local HTTP API for managing tunnels programmatically",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Usage: "Host address to bind the app server",
				Value: appserver.DefaultHost,
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Port to bind the app server",
				Value: appserver.DefaultPort,
			},
			&cli.StringFlag{
				Name:    "token",
				Usage:   "Bearer token required by the app-server API",
				EnvVars: []string{"PORTR_APP_SERVER_TOKEN"},
			},
		},
		Action: startAppServer,
	}
}

func requiresAppServerToken(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || strings.EqualFold(host, "localhost") {
		return false
	}

	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip != nil {
		return !ip.IsLoopback()
	}

	return true
}

func startAppServer(c *cli.Context) error {
	cfg, err := config.Load(c.String("config"))
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	host := c.String("host")
	token := strings.TrimSpace(c.String("token"))
	if requiresAppServerToken(host) && token == "" {
		return fmt.Errorf("--token or PORTR_APP_SERVER_TOKEN is required when binding app-server to %s", host)
	}

	manager := appserver.NewManager(cfg, db.New(&cfg))
	api := appserver.NewServer(manager, token)
	addr := fmt.Sprintf("%s:%d", host, c.Int("port"))

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("Portr app server listening on http://%s\n", addr)
		if err := api.App().Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case <-signalCh:
	case err := <-errCh:
		return err
	case <-c.Context.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	manager.Shutdown(shutdownCtx)
	return api.App().ShutdownWithContext(shutdownCtx)
}
