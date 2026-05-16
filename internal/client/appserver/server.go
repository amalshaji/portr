package appserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type tunnelService interface {
	StartTunnel(context.Context, StartTunnelRequest) (TunnelStatus, error)
	ListTunnels() []TunnelStatus
	GetTunnel(string) (TunnelStatus, error)
	StopTunnel(context.Context, string) (TunnelStatus, error)
	Events(string) []TunnelEvent
}

type Server struct {
	app     *fiber.App
	service tunnelService
	token   string
}

func NewServer(service tunnelService, token string) *Server {
	server := &Server{
		service: service,
		token:   strings.TrimSpace(token),
	}
	server.app = server.buildApp()
	return server
}

func (s *Server) App() *fiber.App {
	return s.app
}

func (s *Server) buildApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(recover.New())
	app.Use(s.authMiddleware)

	app.Get("/api/v1/health", s.handleHealth)
	app.Get("/api/v1/tunnels", s.handleListTunnels)
	app.Post("/api/v1/tunnels", s.handleCreateTunnel)
	app.All("/api/v1/tunnels", methodNotAllowed(fiber.MethodGet, fiber.MethodPost))
	app.Get("/api/v1/tunnels/:id", s.handleGetTunnel)
	app.Delete("/api/v1/tunnels/:id", s.handleStopTunnel)
	app.All("/api/v1/tunnels/:id", methodNotAllowed(fiber.MethodGet, fiber.MethodDelete))
	app.Post("/api/v1/tunnels/:id/shutdown", s.handleStopTunnel)
	app.All("/api/v1/tunnels/:id/shutdown", methodNotAllowed(fiber.MethodPost))
	app.Get("/api/v1/events", s.handleEvents)
	app.All("/api/v1/events", methodNotAllowed(fiber.MethodGet))

	return app
}

func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (s *Server) handleListTunnels(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"tunnels": s.service.ListTunnels()})
}

func (s *Server) handleCreateTunnel(c *fiber.Ctx) error {
	var request StartTunnelRequest
	if err := c.BodyParser(&request); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid JSON body")
	}

	status, err := s.service.StartTunnel(c.UserContext(), request)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(status)
}

func (s *Server) handleGetTunnel(c *fiber.Ctx) error {
	status, err := s.service.GetTunnel(c.Params("id"))
	if err != nil {
		return writeServiceError(c, err)
	}
	return c.JSON(status)
}

func (s *Server) handleStopTunnel(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 10*time.Second)
	defer cancel()

	status, err := s.service.StopTunnel(ctx, c.Params("id"))
	if err != nil {
		return writeServiceError(c, err)
	}
	return c.JSON(status)
}

func (s *Server) handleEvents(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"events": s.service.Events(c.Query("tunnel_id")),
	})
}

func (s *Server) authMiddleware(c *fiber.Ctx) error {
	if s.token == "" {
		return c.Next()
	}

	if c.Get("Authorization") != "Bearer "+s.token {
		return writeError(c, fiber.StatusUnauthorized, "unauthorized")
	}
	return c.Next()
}

func writeServiceError(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrTunnelNotFound) {
		return writeError(c, fiber.StatusNotFound, err.Error())
	}
	return writeError(c, fiber.StatusInternalServerError, err.Error())
}

func writeError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"message": message})
}

func methodNotAllowed(methods ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderAllow, strings.Join(methods, ", "))
		return writeError(c, fiber.StatusMethodNotAllowed, fmt.Sprintf("method must be one of: %s", strings.Join(methods, ", ")))
	}
}
