package handler

import (
	"log/slog"

	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	config  *config.AdminConfig
	service *service.Service
	log     *slog.Logger
}

func New(config *config.AdminConfig, service *service.Service) *Handler {
	return &Handler{config: config, service: service, log: utils.GetLogger()}
}

func (h *Handler) RegisterUserRoutes(app *fiber.App) {
	userGroup := app.Group("/api/user")
	userGroup.Get("/", h.ListUsers)
	userGroup.Get("/me", h.Me)
	userGroup.Patch("/me/update", h.MeUpdate)
	userGroup.Post("/me/logout", h.Logout)
}

func (h *Handler) RegisterConnectionRoutes(app *fiber.App) {
	connectionGroup := app.Group("/api/connection")
	connectionGroup.Get("/", h.ListConnections)
}

func (h *Handler) RegisterGithubAuthRoutes(app *fiber.App) {
	connectionGroup := app.Group("/auth/github")
	connectionGroup.Get("/", h.StartGithubAuth)
	connectionGroup.Get("/callback", h.GithubAuthCallback)
	connectionGroup.Get("/is-superuser-signup", h.IsSuperUserSignup)
}

func (h *Handler) RegisterSettingsRoutes(app *fiber.App) {
	connectionGroup := app.Group("/api/setting")
	connectionGroup.Get("/signup", h.ListSettingsForSignupPage)
	connectionGroup.Get("/all", h.ListSettings)
	connectionGroup.Patch("/signup/update", h.UpdateSignupSettings)
	connectionGroup.Patch("/email/update", h.UpdateEmailSettings)
}

func (h *Handler) RegisterInviteRoutes(app *fiber.App) {
	connectionGroup := app.Group("/api/invite")
	connectionGroup.Get("/", h.ListInvites)
	connectionGroup.Post("/", h.CreateInvite)
}
