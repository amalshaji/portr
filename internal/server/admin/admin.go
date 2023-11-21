package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type AdminServer struct {
	app    *fiber.App
	config *config.AdminConfig
	log    *slog.Logger
}

func New(config *config.AdminConfig) *AdminServer {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	if !config.UseVite {
		app.Static("/", "./build")
	} else {
		app.Use(cors.New(cors.Config{
			AllowOrigins: "http://localhost:5173",
			AllowHeaders: "Origin, Content-Type, Accept",
		}))
	}

	app.Use(logger.New())
	app.Use(recover.New())

	return &AdminServer{
		app:    app,
		config: config,
		log:    utils.GetLogger(),
	}
}

func (s *AdminServer) Start() {
	s.log.Info("starting admin server", "port", s.config.Address())

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.app.Listen(s.config.Address()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("failed to start admin server", "error", err)
			done <- nil
		}
	}()

	<-done
	s.log.Info("stopping admin server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.app.ShutdownWithContext(ctx); err != nil {
		s.log.Error("failed to stop proxy server", "error", err)
	}
}
