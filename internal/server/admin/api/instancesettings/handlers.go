package instancesettings

import (
	"errors"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/amalshaji/portr/internal/server/admin/services"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	autoSignup *services.AutoSignupService
	config     *serverConfig.AdminConfig
}

func NewHandler(db *gorm.DB, cfg *serverConfig.AdminConfig) *Handler {
	return &Handler{
		autoSignup: services.NewAutoSignupService(db),
		config:     cfg,
	}
}

type Response struct {
	GitHubAuthEnabled bool                       `json:"github_auth_enabled"`
	AutoSignupEnabled bool                       `json:"auto_signup_enabled"`
	AutoSignupDomains []AutoSignupDomainResponse `json:"auto_signup_domains"`
}

type AutoSignupDomainResponse struct {
	ID     uint          `json:"id"`
	Domain string        `json:"domain"`
	TeamID uint          `json:"team_id"`
	Team   *TeamResponse `json:"team,omitempty"`
}

type TeamResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type UpdateInput struct {
	AutoSignupEnabled bool                    `json:"auto_signup_enabled"`
	AutoSignupDomains []AutoSignupDomainInput `json:"auto_signup_domains"`
}

type AutoSignupDomainInput struct {
	Domain string `json:"domain"`
	TeamID uint   `json:"team_id"`
}

func (h *Handler) Get(c *fiber.Ctx) error {
	settings, err := h.autoSignup.GetSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load instance settings",
		})
	}

	return c.JSON(h.response(settings))
}

func (h *Handler) Update(c *fiber.Ctx) error {
	var input UpdateInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	if input.AutoSignupEnabled && !h.githubAuthEnabled() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "GitHub authentication must be configured before enabling auto signup",
		})
	}

	settings, err := h.autoSignup.UpdateSettings(input.AutoSignupEnabled, autoSignupDomainInputs(input.AutoSignupDomains))
	if err != nil {
		var validationErr services.AutoSignupValidationError
		if errors.As(err, &validationErr) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": validationErr.Message,
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update instance settings",
		})
	}

	return c.JSON(h.response(settings))
}

func (h *Handler) githubAuthEnabled() bool {
	return h.config != nil && h.config.GithubClientID != "" && h.config.GithubSecret != ""
}

func autoSignupDomainInputs(input []AutoSignupDomainInput) []services.AutoSignupDomainInput {
	domains := make([]services.AutoSignupDomainInput, 0, len(input))
	for _, item := range input {
		domains = append(domains, services.AutoSignupDomainInput{
			Domain: item.Domain,
			TeamID: item.TeamID,
		})
	}
	return domains
}

func (h *Handler) response(settings *services.AutoSignupSettings) Response {
	return Response{
		GitHubAuthEnabled: h.githubAuthEnabled(),
		AutoSignupEnabled: settings.Settings.AutoSignupEnabled,
		AutoSignupDomains: autoSignupDomainResponses(settings.Domains),
	}
}

func autoSignupDomainResponses(domains []models.AutoSignupDomain) []AutoSignupDomainResponse {
	responses := make([]AutoSignupDomainResponse, 0, len(domains))
	for _, domain := range domains {
		var team *TeamResponse
		if domain.Team.ID != 0 {
			team = &TeamResponse{
				ID:   domain.Team.ID,
				Name: domain.Team.Name,
				Slug: domain.Team.Slug,
			}
		}
		responses = append(responses, AutoSignupDomainResponse{
			ID:     domain.ID,
			Domain: domain.Domain,
			TeamID: domain.TeamID,
			Team:   team,
		})
	}

	return responses
}
