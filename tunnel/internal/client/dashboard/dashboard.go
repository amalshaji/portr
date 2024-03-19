package dashboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/dashboard/handler"
	"github.com/amalshaji/portr/internal/client/dashboard/service"
	"github.com/amalshaji/portr/internal/client/dashboard/ui/dist"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/vite"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/django/v3"
)

type Dashboard struct {
	app    *fiber.App
	config *config.Config
	db     *db.Db
	logger *slog.Logger
	port   int
}

func New(db *db.Db, config *config.Config) *Dashboard {
	engine := django.New("./internal/client/dashboard/templates", ".html")
	engine.SetAutoEscape(false)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Views:                 engine,
	})

	app.Use(recover.New())

	if config.UseVite {
		app.Static("/", "./internal/server/admin/web/dist")
		app.Static("/static", "./internal/client/dashboard/static")
	} else {
		app.Use("/static", filesystem.New(filesystem.Config{
			Root:       http.FS(dist.EmbededDirStatic),
			PathPrefix: "static",
		}))
	}

	rootTemplateView := func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"UseVite":  config.UseVite,
			"ViteTags": vite.GenerateViteTags(constants.ClientUiViteDistDir),
		})
	}

	service := service.New(db, config)
	handler := handler.New(config, service)

	app.Get("/", rootTemplateView)
	app.Get("/:id", rootTemplateView)

	tunnelsGroup := app.Group("/api/tunnels")
	handler.RegisterTunnelRoutes(tunnelsGroup)

	return &Dashboard{
		app:    app,
		config: config,
		db:     db,
		logger: utils.GetLogger(),
		port:   7777,
	}
}

func (d *Dashboard) Start() {
	fmt.Println("Dashboard running on http://localhost:7777")

	if err := d.app.Listen(":" + fmt.Sprint(d.port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		if d.config.Debug {
			d.logger.Error("failed to start dashboard server", "error", err)
		}
		os.Exit(1)
	}
}

func (d *Dashboard) Shutdown() {
	if d.config.Debug {
		d.logger.Info("stopping dashboard server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer func() { cancel() }()

	if err := d.app.ShutdownWithContext(ctx); err != nil {
		if d.config.Debug {
			d.logger.Error("failed to stop dashboard server", "error", err)
		}
		os.Exit(1)
	}
}
