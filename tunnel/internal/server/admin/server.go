package admin

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/amalshaji/portr/internal/server/admin/api/auth"
	"github.com/amalshaji/portr/internal/server/admin/api/config"
	"github.com/amalshaji/portr/internal/server/admin/api/connection"
	"github.com/amalshaji/portr/internal/server/admin/api/team"
	"github.com/amalshaji/portr/internal/server/admin/api/user"
	"github.com/amalshaji/portr/internal/server/admin/db"
	"github.com/amalshaji/portr/internal/server/admin/middleware"
	"github.com/amalshaji/portr/internal/server/admin/scheduler"
	"github.com/amalshaji/portr/internal/server/admin/utils"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/stats"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
	"gorm.io/gorm"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	app            *fiber.App
	config         *serverConfig.AdminConfig
	db             *db.AdminDB
	auth           *middleware.AuthMiddleware
	scheduler      *scheduler.Scheduler
	store          *session.Store
	startTime      time.Time
	statsCollector *stats.StatsCollector
}

func NewServer(cfg *serverConfig.AdminConfig, database *gorm.DB) *Server {
	engine := html.NewFileSystem(http.FS(templateFS), ".html")
	engine.Reload(cfg.Debug)

	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
		ErrorHandler:      errorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.DomainAddress(),
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Team-Slug",
		AllowCredentials: true,
	}))

	// Create session store
	store := session.New()

	server := &Server{
		app:            app,
		config:         cfg,
		db:             db.New(database),
		auth:           middleware.NewAuthMiddleware(database),
		scheduler:      scheduler.New(database),
		store:          store,
		startTime:      time.Now(),
		statsCollector: stats.NewStatsCollector(),
	}

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("server_start_time", server.startTime)
		return c.Next()
	})

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	api := s.app.Group("/api")
	v1 := api.Group("/v1")

	v1.Get("/healthcheck", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	s.setupAuthRoutes(v1)
	s.setupUserRoutes(v1)
	s.setupTeamRoutes(v1)
	s.setupConnectionRoutes(v1)
	s.setupConfigRoutes(v1)

	s.setupInstanceSettingsRoutes(v1)

	s.app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticFS),
		PathPrefix: "static",
		Browse:     false,
	}))

	s.app.Get("/*", s.auth.RequireAuthRedirect, func(c *fiber.Ctx) error {
		// Skip static paths
		if strings.HasPrefix(c.Path(), "/static/") {
			return c.Next()
		}
		return s.handleIndex(c)
	})
}

func (s *Server) setupAuthRoutes(v1 fiber.Router) {
	authHandler := auth.NewHandler(s.db.DB, s.store, s.config)
	authGroup := v1.Group("/auth")

	authGroup.Get("/auth-config", authHandler.GetAuthConfig)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/logout", s.auth.RequireAuth, authHandler.Logout)

	authGroup.Get("/github", authHandler.GitHubLogin)
	authGroup.Get("/github/callback", authHandler.GitHubCallback)
}

func (s *Server) setupUserRoutes(v1 fiber.Router) {
	userHandler := user.NewHandler(s.db.DB, s.store)
	userGroup := v1.Group("/user")

	userGroup.Get("/me", s.auth.RequireTeamUser, userHandler.GetCurrentUser)
	userGroup.Get("/me/teams", s.auth.RequireAuth, userHandler.GetUserTeams)
	userGroup.Patch("/me/update", s.auth.RequireAuth, userHandler.UpdateUser)
	userGroup.Patch("/me/change-password", s.auth.RequireAuth, userHandler.ChangePassword)
	userGroup.Patch("/me/rotate-secret-key", s.auth.RequireTeamUser, userHandler.RotateSecretKey)
}

func (s *Server) setupTeamRoutes(v1 fiber.Router) {
	teamHandler := team.NewHandler(s.db.DB, s.store)
	teamGroup := v1.Group("/team")

	teamGroup.Post("/", s.auth.RequireSuperuser, teamHandler.CreateTeam)
	teamGroup.Get("/users", s.auth.RequireTeamUser, teamHandler.GetTeamUsers)
	teamGroup.Post("/add", s.auth.RequireAdmin, teamHandler.AddUser)
	teamGroup.Delete("/users/:id", s.auth.RequireAdmin, teamHandler.RemoveUser)
	teamGroup.Post("/users/:id/reset-password", s.auth.RequireAdmin, teamHandler.ResetUserPassword)
}

func (s *Server) setupConnectionRoutes(v1 fiber.Router) {
	connHandler := connection.NewHandler(s.db.DB, s.store)
	connGroup := v1.Group("/connections")

	connGroup.Get("/", s.auth.RequireTeamUser, connHandler.GetConnections)
	connGroup.Post("/", connHandler.CreateConnection)
}

func (s *Server) setupConfigRoutes(v1 fiber.Router) {
	configHandler := config.NewHandler(s.db.DB, s.store, s.config, s.statsCollector)
	configGroup := v1.Group("/config")

	configGroup.Post("/download", configHandler.DownloadConfig)
	configGroup.Get("/setup-script", s.auth.RequireTeamUser, configHandler.GetSetupScript)
	configGroup.Get("/stats", s.auth.RequireTeamUser, configHandler.GetStats)
}

func (s *Server) setupInstanceSettingsRoutes(v1 fiber.Router) {
	configHandler := config.NewHandler(s.db.DB, s.store, s.config, s.statsCollector)
	instanceGroup := v1.Group("/instance-settings")

	instanceGroup.Get("/", s.auth.RequireSuperuser, configHandler.GetInstanceSettings)
	instanceGroup.Patch("/", s.auth.RequireSuperuser, configHandler.UpdateInstanceSettings)
}

func (s *Server) handleIndex(c *fiber.Ctx) error {
	data := fiber.Map{
		"UseVite":  s.config.UseVite,
		"ViteTags": s.generateViteTags(),
	}

	return c.Render("templates/index", data)
}

func (s *Server) generateViteTags() template.HTML {
	if s.config.UseVite {
		return ""
	}

	// Read the Vite manifest from the embedded static filesystem and
	// pass the bytes to the utils helper which generates the tags.
	manifestBytes, err := staticFS.ReadFile("static/.vite/manifest.json")
	if err != nil {
		// If we can't read the manifest (not present), fall back to empty tags.
		return ""
	}

	return utils.GenerateViteTagsFromBytes(manifestBytes)
}

func (s *Server) App() *fiber.App {
	return s.app
}

func (s *Server) Store() *session.Store {
	return s.store
}

func (s *Server) Start() error {
	s.scheduler.Start()
	s.statsCollector.Start()

	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Info("Starting admin server", "address", addr)
	return s.app.Listen(addr)
}

func (s *Server) Shutdown() error {
	s.scheduler.Stop()
	s.statsCollector.Stop()

	return s.app.Shutdown()
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	log.Error("Request error", "error", err, "status_code", code)

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
