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

func (h *Handler) RegisterTeamUserRoutes(group fiber.Router, permissionHandler fiber.Handler) {
	userGroup := group.Group("/user")
	userGroup.Get("/", h.ListTeamUsers)
	userGroup.Post("/add", h.AddMember)
	userGroup.Get("/me", h.MeInTeam)
	userGroup.Patch("/me/rotate-secret-key", permissionHandler, h.RotateSecretKey)
}

func (h *Handler) RegisterUserRoutes(group fiber.Router) {
	currentUserGroup := group.Group("/user")
	currentUserGroup.Get("/me", h.Me)
	currentUserGroup.Patch("/me/update", h.MeUpdate)
	currentUserGroup.Post("/me/logout", h.Logout)
}

func (h *Handler) RegisterConnectionRoutes(group fiber.Router) {
	connectionGroup := group.Group("/connection")
	connectionGroup.Get("/", h.ListConnections)
}

func (h *Handler) RegisterGithubAuthRoutes(group fiber.Router) {
	group.Get("/", h.StartGithubAuth)
	group.Get("/callback", h.GithubAuthCallback)
	group.Get("/is-superuser-signup", h.IsSuperUserSignup)
}

func (h *Handler) RegisterSettingsRoutes(group fiber.Router, permissionHandler fiber.Handler) {
	settingsGroup := group.Group("/setting", permissionHandler)
	settingsGroup.Get("/all", h.ListSettings)
	settingsGroup.Patch("/email/update", h.UpdateEmailSettings)
}

func (h *Handler) RegisterClientConfigRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	configGroup := app.Group("/config")
	configGroup.Post("/validate", h.ValidateClientConfig)
	configGroup.Get("/address", authMiddleware, h.GetServerAddress)
}

func (h *Handler) RegisterTeamRoutes(
	group fiber.Router,
	permissionHandler fiber.Handler,
) {
	teamGroup := group.Group("/team", permissionHandler)
	teamGroup.Post("/", h.CreateTeam)
}
