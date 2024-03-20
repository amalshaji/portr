package handler

import (
	"log/slog"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/dashboard/service"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	config  *config.Config
	service *service.Service
	log     *slog.Logger
}

func New(config *config.Config, service *service.Service) *Handler {
	return &Handler{
		config:  config,
		service: service,
		log:     utils.GetLogger(),
	}
}

func (h *Handler) RegisterTunnelRoutes(group fiber.Router) {
	group.Get("/", h.GetTunnels)
	group.Get("/render/:id", h.RenderResponse)
	group.Get("/replay/:id", h.ReplayRequest)
	group.Get("/:subdomain/:port", h.GetRequests)
}
