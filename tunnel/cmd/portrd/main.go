package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/amalshaji/portr/internal/server/admin"
	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/cron"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/service"
	sshd "github.com/amalshaji/portr/internal/server/ssh"
	"github.com/charmbracelet/log"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/urfave/cli/v2"
)

// Set at build time
var version = "0.0.0"

func main() {
	app := &cli.App{
		Name:    "portrd",
		Usage:   "portr server",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start servers",
				Subcommands: []*cli.Command{
					{
						Name:  "tunnel",
						Usage: "Start tunnel server only",
						Action: func(c *cli.Context) error {
							startTunnel(c.String("config"))
							return nil
						},
					},
					{
						Name:  "admin",
						Usage: "Start admin server only",
						Action: func(c *cli.Context) error {
							return startAdmin()
						},
					},
					{
						Name:  "all",
						Usage: "Start both tunnel and admin servers",
						Action: func(c *cli.Context) error {
							return startAll(c.String("config"))
						},
					},
				},
			},
			{
				Name:  "migrate",
				Usage: "Run database migrations",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "dialect",
						Usage:    "Database dialect (postgres or sqlite)",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return runMigrations(c.String("dialect"))
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal("Failed to run application", "error", err)
	}
}

func runMigrations(dialect string) error {
	if dialect != "postgres" && dialect != "sqlite" {
		return fmt.Errorf("unsupported dialect: %s (supported: postgres, sqlite)", dialect)
	}

	cfg := config.Load("")
	dbUrl := cfg.Database.Url
	driver := cfg.Database.Driver

	// Map driver to expected dialect and prepare connection
	switch driver {
	case "sqlite3", "sqlite":
		if dialect != "sqlite" {
			return fmt.Errorf("database driver is %s but dialect is %s", driver, dialect)
		}
		driver = "sqlite3"
		if strings.Contains(dbUrl, "://") {
			parts := strings.Split(dbUrl, "://")
			if len(parts) > 1 {
				dbUrl = parts[1]
			}
		}
	case "postgres", "postgresql":
		if dialect != "postgres" {
			return fmt.Errorf("database driver is %s but dialect is %s", driver, dialect)
		}
		driver = "postgres"
		// Add sslmode=disable if not specified
		if !strings.Contains(dbUrl, "sslmode=") {
			separator := "?"
			if strings.Contains(dbUrl, "?") {
				separator = "&"
			}
			dbUrl = dbUrl + separator + "sslmode=disable"
		}
	default:
		return fmt.Errorf("unsupported database driver: %s", driver)
	}

	db, err := sql.Open(driver, dbUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set goose dialect
	goose.SetDialect(driver)

	// Set migration directory based on dialect
	migrationDir := fmt.Sprintf("migrations/%s", dialect)

	// Run migrations
	if err := goose.Up(db, migrationDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info("Migrations completed successfully")
	return nil
}

// runAutoMigrations runs migrations if AutoMigrate is enabled in config
func runAutoMigrations(cfg *config.Config) error {
	if !cfg.Database.AutoMigrate {
		return nil
	}

	log.Info("Running auto-migrations...")

	// Map database driver to dialect
	var dialect string
	switch cfg.Database.Driver {
	case "sqlite3", "sqlite":
		dialect = "sqlite"
	case "postgres", "postgresql":
		dialect = "postgres"
	default:
		return fmt.Errorf("unsupported database driver for auto-migration: %s", cfg.Database.Driver)
	}

	return runMigrations(dialect)
}

func startTunnel(configFilePath string) {
	config := config.Load(configFilePath)

	// Run auto-migrations if enabled
	if err := runAutoMigrations(config); err != nil {
		log.Fatal("Failed to run auto-migrations", "error", err)
	}

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

func startAdmin() error {
	// Load configuration
	fullConfig := config.Load("")
	cfg := &fullConfig.Admin

	// Run auto-migrations if enabled
	if err := runAutoMigrations(fullConfig); err != nil {
		return fmt.Errorf("failed to run auto-migrations: %w", err)
	}

	_db := db.New(&fullConfig.Database)
	_db.Connect()

	adminServer := admin.NewServer(cfg, _db.Conn)

	// Handle shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := adminServer.Start(); err != nil {
			log.Fatal("Failed to start admin server", "error", err)
		}
	}()

	<-done

	log.Info("Shutting down admin server...")
	return adminServer.Shutdown()
}

func startAll(configFilePath string) error {
	// Load configurations
	tunnelConfig := config.Load(configFilePath)
	adminCfg := &tunnelConfig.Admin

	// Run auto-migrations if enabled
	if err := runAutoMigrations(tunnelConfig); err != nil {
		return fmt.Errorf("failed to run auto-migrations: %w", err)
	}

	_db := db.New(&tunnelConfig.Database)
	_db.Connect()

	service := service.New(_db)
	proxyServer := proxy.New(tunnelConfig)
	sshServer := sshd.New(&tunnelConfig.Ssh, proxyServer, service)
	cronJob := cron.New(_db, tunnelConfig, service)
	adminServer := admin.NewServer(adminCfg, _db.Conn)

	// Use WaitGroup to track all servers
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Start tunnel components
	wg.Add(3)
	go func() {
		defer wg.Done()
		proxyServer.Start()
	}()

	go func() {
		defer wg.Done()
		sshServer.Start()
	}()

	go func() {
		defer wg.Done()
		cronJob.Start()
	}()

	// Start admin server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := adminServer.Start(); err != nil {
			log.Error("Admin server error", "error", err)
			cancel()
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-done:
		log.Info("Received shutdown signal")
	case <-ctx.Done():
		log.Info("Context cancelled due to error")
	}

	// Shutdown all services
	log.Info("Shutting down all services...")

	var shutdownErr error

	// Shutdown admin server
	if err := adminServer.Shutdown(); err != nil {
		log.Error("Error shutting down admin server", "error", err)
		shutdownErr = err
	}

	// Shutdown tunnel components
	proxyServer.Shutdown(context.TODO())
	sshServer.Shutdown(context.TODO())
	cronJob.Shutdown()

	// Wait for all goroutines to finish
	cancel()
	wg.Wait()

	if shutdownErr != nil {
		return fmt.Errorf("shutdown completed with errors: %w", shutdownErr)
	}

	log.Info("All services shut down successfully")
	return nil
}
