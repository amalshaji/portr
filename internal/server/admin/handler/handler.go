package handler

import (
	"log/slog"

	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	config  *config.Config
	service *service.Service
	log     *slog.Logger
}

func New(config *config.Config, service *service.Service) *Handler {
	return &Handler{config: config, service: service, log: utils.GetLogger()}
}

func (h *Handler) RegisterUserRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	userGroup := app.Group("/api/user", authMiddleware)
	userGroup.Get("/", h.ListUsers)
	userGroup.Get("/me", h.Me)
	userGroup.Patch("/me/update", h.MeUpdate)
	userGroup.Patch("/me/rotate-secret-key", h.RotateSecretKey)
	userGroup.Post("/me/logout", h.Logout)
}

func (h *Handler) RegisterConnectionRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	connectionGroup := app.Group("/api/connection", authMiddleware)
	connectionGroup.Get("/", h.ListConnections)
}

func (h *Handler) RegisterGithubAuthRoutes(app *fiber.App) {
	githubAuthGroup := app.Group("/auth/github")
	githubAuthGroup.Get("/", h.StartGithubAuth)
	githubAuthGroup.Get("/callback", h.GithubAuthCallback)
	githubAuthGroup.Get("/is-superuser-signup", h.IsSuperUserSignup)
}

func (h *Handler) RegisterSettingsRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	settingsGroup := app.Group("/api/setting")
	settingsGroup.Get("/signup", h.ListSettingsForSignupPage)
	settingsGroup.Get("/all", authMiddleware, h.ListSettings)
	settingsGroup.Patch("/signup/update", authMiddleware, h.UpdateSignupSettings)
	settingsGroup.Patch("/email/update", authMiddleware, h.UpdateEmailSettings)
}

func (h *Handler) RegisterInviteRoutes(
	app *fiber.App,
	authMiddleware fiber.Handler,
	permissionHandler fiber.Handler,
) {
	inviteGroup := app.Group("/api/invite", authMiddleware)
	inviteGroup.Get("/", h.ListInvites)
	inviteGroup.Post("/", permissionHandler, h.CreateInvite)

	inviteAcceptGroup := app.Group("/invite")
	inviteAcceptGroup.Get("/:code", h.AcceptInvite)
}

func (h *Handler) RegisterClientConfigRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	configGroup := app.Group("/config")
	configGroup.Post("/validate", h.ValidateClientConfig)
	configGroup.Get("/address", authMiddleware, h.GetServerAddress)
}
