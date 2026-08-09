package config

import (
	"strings"

	clientConfig "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/server/admin/middleware"
	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/gofiber/fiber/v2"
)

type ClientTemplateInput struct {
	Template string `json:"template"`
}

func (h *Handler) GetClientTemplate(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	return c.JSON(clientTemplateResponse(teamUser.Team))
}

func (h *Handler) UpdateClientTemplate(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	var input ClientTemplateInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	template := strings.TrimSpace(input.Template)
	if err := clientConfig.ValidateTemplate(template); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.db.Model(&teamUser.Team).Update("client_template", template).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save client template",
		})
	}

	return c.JSON(clientTemplateResponse(teamUser.Team))
}

func clientTemplateResponse(team models.Team) fiber.Map {
	return fiber.Map{
		"template":   team.ClientTemplate,
		"updated_at": team.UpdatedAt,
	}
}
