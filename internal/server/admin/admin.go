package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amalshaji/localport/internal/server/admin/handler"
	"github.com/amalshaji/localport/internal/server/admin/service"

	"github.com/amalshaji/localport/internal/server/config"
	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/django/v3"
)

type AdminServer struct {
	app    *fiber.App
	config *config.AdminConfig
	log    *slog.Logger
}

func New(config *config.Config, service *service.Service) *AdminServer {
	engine := django.New("./internal/server/admin/templates", ".html")
	engine.SetAutoEscape(false)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Views:                 engine,
	})

	app.Use(logger.New())
	app.Use(recover.New())

	ctx := context.Background()

	if !config.Admin.UseVite {
		app.Static("/", "./internal/server/admin/web/dist")
	}

	app.Static("/static", "./internal/server/admin/static", fiber.Static{
		Compress: true,
	})

	clientPages := []string{"/connections", "/overview", "/settings", "/users", "/my-account", "/new-team"}

	app.Use(func(c *fiber.Ctx) error {
		token := c.Cookies("localport-session")
		user, _ := service.GetUserBySession(ctx, token)

		c.Locals("user", user)
		return c.Next()
	})

	// set teamUser in locals
	gleanTeamUser := func(c *fiber.Ctx) error {
		user := c.Locals("user").(*db.UserWithTeams)
		if user == nil {
			return c.Next()
		}
		teamName := c.Params("teamName")
		teamUser, _ := service.GetTeamUser(ctx, user.ID, teamName)
		c.Locals("teamUser", teamUser)
		return c.Next()
	}

	handler := handler.New(config, service)

	githubAuthGroup := app.Group("/auth/github")
	handler.RegisterGithubAuthRoutes(githubAuthGroup)

	connectionForClientGroup := app.Group("/api/")
	handler.RegisterConnectionRoutesForClient(connectionForClientGroup)

	apiGroup := app.Group("/api/", apiAuthMiddleware)
	handler.RegisterUserRoutes(apiGroup)
	handler.RegisterSettingsRoutes(apiGroup, superUserPermissionRequired)
	handler.RegisterTeamRoutes(apiGroup, superUserPermissionRequired)

	handler.RegisterClientConfigRoutes(app, apiAuthMiddleware)

	teamApiGroup := app.Group("/api/:teamName", gleanTeamUser, apiTeamAuthMiddleware)
	handler.RegisterTeamUserRoutes(teamApiGroup, adminPermissionRequired)
	handler.RegisterConnectionRoutes(teamApiGroup)

	// handle initial setup
	app.Use(func(c *fiber.Ctx) error {
		user := c.Locals("user").(*db.UserWithTeams)
		if user != nil && len(user.Teams) == 0 && c.Path() != "/setup" {
			return c.Redirect("/setup")
		}
		if user != nil && len(user.Teams) > 0 && c.Path() == "/setup" {
			return c.Redirect(fmt.Sprintf("/%s/overview", user.Teams[0].Name))
		}
		return c.Next()
	})

	// server index templates for all routes
	// should be explicit?
	rootTemplateView := func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"UseVite":  config.Admin.UseVite,
			"ViteTags": getViteTags(),
		})
	}

	app.Get("/", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*db.UserWithTeams)
		if user != nil {
			if len(user.Teams) == 0 {
				return c.Redirect("/setup")
			}
			return c.Redirect(fmt.Sprintf("/%s/overview", user.Teams[0].Name))
		}

		return rootTemplateView(c)
	})

	app.Get("/setup", rootViewAuthMiddleware, rootTemplateView)

	for _, page := range clientPages {
		app.Get("/:teamName"+page, gleanTeamUser, teamViewAuthMiddleware, rootTemplateView)
	}

	return &AdminServer{
		app:    app,
		config: &config.Admin,
		log:    utils.GetLogger(),
	}
}

func (s *AdminServer) Start() {
	s.log.Info("starting admin server", "port", s.config.ListenAddress())

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.app.Listen(s.config.ListenAddress()); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
