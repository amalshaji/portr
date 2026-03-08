package dashboard

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/dashboard/handler"
	"github.com/amalshaji/portr/internal/client/dashboard/service"
	"github.com/amalshaji/portr/internal/client/dashboard/ui/dist"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/vite"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/django/v3"
)

type Dashboard struct {
	app    *fiber.App
	config *config.Config
	db     *db.Db
	port   int
}

//go:embed templates
var templatesFS embed.FS

func New(db *db.Db, config *config.Config) *Dashboard {
	var engine *django.Engine

	if config.UseVite {
		engine = django.New("./internal/client/dashboard/templates", ".html")
	} else {
		engine = django.NewPathForwardingFileSystem(http.FS(templatesFS), "/templates", ".html")
	}

	engine.SetAutoEscape(false)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Views:                 engine,
	})

	app.Use(recover.New())

	if config.UseVite {
		app.Static("/static", "./internal/client/dashboard/static")
	} else {
		app.Use("/static", filesystem.New(filesystem.Config{
			Root:       http.FS(dist.EmbeddedDirStatic),
			PathPrefix: "static",
		}))
	}

	rootTemplateView := func(c *fiber.Ctx) error {
		context := fiber.Map{
			"UseVite": config.UseVite,
		}
		if !config.UseVite {
			context["ViteTags"] = vite.GenerateViteTags(dist.ManifestString)
		}
		return c.Render("index", context)
	}

	service := service.New(db, config)
	handler := handler.New(config, service)

	app.Get("/is-this-portr-server", func(c *fiber.Ctx) error {
		return c.SendString("yes")
	})
	app.Get("/", rootTemplateView)
	app.Get("/:id", rootTemplateView)

	tunnelsGroup := app.Group("/api/tunnels")
	handler.RegisterTunnelRoutes(tunnelsGroup)

	return &Dashboard{
		app:    app,
		config: config,
		db:     db,
		port:   7777,
	}
}

func (d *Dashboard) Start() error {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/is-this-portr-server", d.port))
	if err == nil {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err == nil && string(body) == "yes" {
				return nil
			}
		}
	}

	if err := d.app.Listen(":" + fmt.Sprint(d.port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		if d.config.Debug {
			log.Error("Failed to start dashboard server", "error", err)
		}
		return err
	}

	return nil
}

func (d *Dashboard) Shutdown() {
	if d.config.Debug {
		log.Debug("Stopping dashboard server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer func() { cancel() }()

	if err := d.app.ShutdownWithContext(ctx); err != nil {
		if d.config.Debug {
			log.Error("Failed to stop dashboard server", "error", err)
		}
		os.Exit(1)
	}
}
