package admin

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/amalshaji/portr/internal/admin/api/auth"
	"github.com/amalshaji/portr/internal/admin/api/config"
	"github.com/amalshaji/portr/internal/admin/api/connection"
	"github.com/amalshaji/portr/internal/admin/api/team"
	"github.com/amalshaji/portr/internal/admin/api/user"
	adminConfig "github.com/amalshaji/portr/internal/admin/config"
	"github.com/amalshaji/portr/internal/admin/db"
	"github.com/amalshaji/portr/internal/admin/middleware"
	"github.com/amalshaji/portr/internal/admin/scheduler"
	"github.com/amalshaji/portr/internal/admin/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"gorm.io/gorm"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server represents the admin server
type Server struct {
	app    *fiber.App
	config *adminConfig.AdminConfig
	db     *db.AdminDB
	// ...existing code...
	auth      *middleware.AuthMiddleware
	scheduler *scheduler.Scheduler
	startTime time.Time
}

// NewServer creates a new admin server
func NewServer(cfg *adminConfig.AdminConfig, database *gorm.DB) *Server {
	// Initialize template engine
	engine := html.NewFileSystem(http.FS(templateFS), ".html")
	engine.Reload(cfg.Debug)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
		ErrorHandler:      errorHandler,
	})

	// Initialize middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.DomainAddress(),
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Team-Slug",
		AllowCredentials: true,
	}))

	// Create server instance
	server := &Server{
		app:       app,
		config:    cfg,
		db:        db.New(database),
		auth:      middleware.NewAuthMiddleware(database),
		scheduler: scheduler.New(database),
		startTime: time.Now(),
	}

	// Middleware to add server start time to context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("server_start_time", server.startTime)
		return c.Next()
	})

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Serve static files
	s.app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticFS),
		PathPrefix: "static",
		Browse:     false,
	}))

	// API routes
	api := s.app.Group("/api")
	v1 := api.Group("/v1")

	// Health check
	v1.Get("/healthcheck", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Setup API routes
	s.setupAuthRoutes(v1)
	s.setupUserRoutes(v1)
	s.setupTeamRoutes(v1)
	s.setupConnectionRoutes(v1)
	s.setupConfigRoutes(v1)

	// setupInstanceSettingsRoutes configures instance settings routes
	s.setupInstanceSettingsRoutes(v1)

	// Main route - serve the SPA
	s.app.Get("/", s.handleIndex)
	s.app.Get("/*", s.handleIndex) // Catch-all for SPA routing
}

// setupAuthRoutes configures authentication routes
func (s *Server) setupAuthRoutes(v1 fiber.Router) {
	authHandler := auth.NewHandler(s.db.DB, nil, s.config)
	authGroup := v1.Group("/auth")

	authGroup.Get("/auth-config", authHandler.GetAuthConfig)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/logout", s.auth.RequireAuth, authHandler.Logout)

	// GitHub OAuth routes - matching Python implementation
	authGroup.Get("/github", authHandler.GitHubLogin)
	authGroup.Get("/github/callback", authHandler.GitHubCallback)
} // setupUserRoutes configures user routes
func (s *Server) setupUserRoutes(v1 fiber.Router) {
	userHandler := user.NewHandler(s.db.DB, nil)
	userGroup := v1.Group("/user")

	userGroup.Get("/me", s.auth.RequireTeamUser, userHandler.GetCurrentUser)
	userGroup.Get("/me/teams", s.auth.RequireAuth, userHandler.GetUserTeams)
	userGroup.Patch("/me/update", s.auth.RequireAuth, userHandler.UpdateUser)
	userGroup.Patch("/me/change-password", s.auth.RequireAuth, userHandler.ChangePassword)
	userGroup.Patch("/me/rotate-secret-key", s.auth.RequireTeamUser, userHandler.RotateSecretKey)
}

// setupTeamRoutes configures team routes
func (s *Server) setupTeamRoutes(v1 fiber.Router) {
	teamHandler := team.NewHandler(s.db.DB, nil)
	teamGroup := v1.Group("/team")

	teamGroup.Post("/", s.auth.RequireSuperuser, teamHandler.CreateTeam)
	teamGroup.Get("/users", s.auth.RequireTeamUser, teamHandler.GetTeamUsers)
	teamGroup.Post("/add", s.auth.RequireAdmin, teamHandler.AddUser)
	teamGroup.Delete("/users/:id", s.auth.RequireAdmin, teamHandler.RemoveUser)
}

// setupConnectionRoutes configures connection routes
func (s *Server) setupConnectionRoutes(v1 fiber.Router) {
	connHandler := connection.NewHandler(s.db.DB, nil)
	connGroup := v1.Group("/connections")

	connGroup.Get("/", s.auth.RequireTeamUser, connHandler.GetConnections)
	connGroup.Post("/", connHandler.CreateConnection) // Uses secret key auth
}

// setupConfigRoutes configures config routes
func (s *Server) setupConfigRoutes(v1 fiber.Router) {
	configHandler := config.NewHandler(s.db.DB, nil, s.config)
	configGroup := v1.Group("/config")

	configGroup.Post("/download", configHandler.DownloadConfig)
	configGroup.Get("/setup-script", s.auth.RequireTeamUser, configHandler.GetSetupScript)
	configGroup.Get("/stats", s.auth.RequireTeamUser, configHandler.GetStats)
}

// setupInstanceSettingsRoutes configures instance settings routes
func (s *Server) setupInstanceSettingsRoutes(v1 fiber.Router) {
	configHandler := config.NewHandler(s.db.DB, nil, s.config)
	instanceGroup := v1.Group("/instance-settings")

	instanceGroup.Get("/", s.auth.RequireSuperuser, configHandler.GetInstanceSettings)
	instanceGroup.Patch("/", s.auth.RequireSuperuser, configHandler.UpdateInstanceSettings)
}

// handleIndex serves the main HTML template
func (s *Server) handleIndex(c *fiber.Ctx) error {
	// Prepare template data
	fmt.Println("UseVite:", s.config.UseVite)
	data := fiber.Map{
		"UseVite":  s.config.UseVite,
		"ViteTags": s.generateViteTags(),
	}

	// Use the template engine to render the embedded template
	return c.Render("templates/index", data)
}

// generateViteTags generates Vite development server tags or production asset tags
func (s *Server) generateViteTags() string {
	if s.config.UseVite {
		// In development mode with Vite, the script tags are added directly in the template
		return ""
	}

	// In production mode, generate tags from manifest
	return utils.GenerateViteTags()
}

// Start starts the admin server
func (s *Server) Start() error {
	s.scheduler.Start()

	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Printf("Starting admin server on %s", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	// Stop background jobs
	s.scheduler.Stop()

	// Shutdown Fiber server
	return s.app.Shutdown()
}

// errorHandler handles errors globally
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log the error
	log.Printf("Error: %v", err)

	// Return JSON error response
	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
