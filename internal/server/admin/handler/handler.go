package handler

import (
	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/server/config"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	config  *config.AdminConfig
	service *service.Service
}

func New(config *config.AdminConfig, service *service.Service) *Handler {
	return &Handler{config: config, service: service}
}

func (h *Handler) RegisterUserRoutes(app *fiber.App) {
	userGroup := app.Group("/api/users")
	userGroup.Get("/", h.ListUsers)
}

func (h *Handler) RegisterConnectionRoutes(app *fiber.App) {
	connectionGroup := app.Group("/api/connections")
	connectionGroup.Get("/", h.ListActiveConnections)
}
