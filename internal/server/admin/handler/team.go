package handler

import (
	"net/http"

	"github.com/amalshaji/localport/internal/server/admin/service"
	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateTeam(c *fiber.Ctx) error {
	var createTeamInput service.CreateTeamInput
	if err := c.BodyParser(&createTeamInput); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	user := c.Locals("user").(*db.UserWithTeams)
	team, err := h.service.CreateFirstTeam(c.Context(), createTeamInput, user.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(team)
}

func (h *Handler) AddMember(c *fiber.Ctx) error {
	var payload service.AddMemberInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	result, err := h.service.AddMember(c.Context(), payload, teamUser.TeamID, teamUser.ID)
	if err != nil {
		h.log.Error("error while logging out", "error", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(result)
}
