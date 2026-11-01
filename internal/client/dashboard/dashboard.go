package dashboard

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/amalshaji/portr/internal/client/dashboard/handler"
	"github.com/amalshaji/portr/internal/client/dashboard/service"
	"github.com/amalshaji/portr/internal/client/dashboard/ui-v2/dist"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/vite"
	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/utils"
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
	host   string
	port   int
}

//go:embed templates
var templatesFS embed.FS

// New builds the dashboard. host is the interface to listen on; an empty host
// listens on every interface, which only makes sense inside a container.
func New(db *db.Db, config *config.Config, host string) *Dashboard {
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
	app.Use(localOnly)

	if config.UseVite {
		app.Static("/static", "./internal/client/dashboard/ui-v2/public")
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
		host:   host,
		port:   config.DashboardPort,
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

	if err := d.app.Listen(net.JoinHostPort(d.host, fmt.Sprint(d.port))); err != nil && !errors.Is(err, http.ErrServerClosed) {
		if d.config.Debug {
			log.Error("Failed to start dashboard server", "error", err)
		}
		return err
	}

	return nil
}

func (d *Dashboard) Shutdown() error {
	if d.config.Debug {
		log.Debug("Stopping dashboard server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer func() { cancel() }()

	if err := d.app.ShutdownWithContext(ctx); err != nil {
		if d.config.Debug {
			log.Error("Failed to stop dashboard server", "error", err)
		}
		return err
	}

	return nil
}

// localOnly rejects requests that do not come from a page on localhost. The
// dashboard has no auth and exposes captured traffic and replay, so a foreign
// Host (DNS rebinding) or a foreign Origin (cross-site request) is refused. The
// raw Host header is used because fiber's Hostname trusts X-Forwarded-Host.
func localOnly(c *fiber.Ctx) error {
	host := string(c.Request().Host())
	if !utils.IsLoopbackHost(host) {
		return c.Status(fiber.StatusForbidden).SendString("forbidden")
	}
	if origin := c.Get(fiber.HeaderOrigin); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host != host {
			return c.Status(fiber.StatusForbidden).SendString("forbidden")
		}
	}
	return c.Next()
}
