package handler

import (
	"net/http"

	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/server/db"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateTeam(c *fiber.Ctx) error {
	var createTeamInput service.CreateTeamInput
	if err := c.BodyParser(&createTeamInput); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	user := c.Locals("user").(*db.User)
	team, err := h.service.CreateFirstTeam(createTeamInput, user)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(team)
}
