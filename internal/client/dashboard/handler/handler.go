package handler

import (
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/dashboard/service"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	config  *config.Config
	service *service.Service
}

func New(config *config.Config, service *service.Service) *Handler {
	return &Handler{
		config:  config,
		service: service,
	}
}

func (h *Handler) RegisterTunnelRoutes(group fiber.Router) {
	group.Get("/", h.GetTunnels)
	group.Get("/render/:id", h.RenderResponse)
	group.Get("/replay/:id", h.ReplayRequest)
	group.Delete("/:subdomain/:port", h.DeleteRequests)
	group.Get("/:subdomain/:port", h.GetRequests)
}
