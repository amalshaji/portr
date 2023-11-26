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
	githubAuthGroup := app.Group("/auth/github")
	githubAuthGroup.Get("/", h.StartGithubAuth)
	githubAuthGroup.Get("/callback", h.GithubAuthCallback)
	githubAuthGroup.Get("/is-superuser-signup", h.IsSuperUserSignup)
}

func (h *Handler) RegisterSettingsRoutes(app *fiber.App) {
	settingsGroup := app.Group("/api/setting")
	settingsGroup.Get("/signup", h.ListSettingsForSignupPage)
	settingsGroup.Get("/all", h.ListSettings)
	settingsGroup.Patch("/signup/update", h.UpdateSignupSettings)
	settingsGroup.Patch("/email/update", h.UpdateEmailSettings)
}

func (h *Handler) RegisterInviteRoutes(app *fiber.App) {
	inviteGroup := app.Group("/api/invite")
	inviteGroup.Get("/", h.ListInvites)
	inviteGroup.Post("/", h.CreateInvite)

	inviteAcceptGroup := app.Group("/invite")
	inviteAcceptGroup.Get("/:code", h.AcceptInvite)
}
