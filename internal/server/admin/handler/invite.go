package handler

import (
	"net/http"

	"github.com/amalshaji/localport/internal/server/admin/service"
	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateInvite(c *fiber.Ctx) error {
	var payload service.CreateInviteInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	user, err := h.service.CreateInvite(c.Context(), payload, teamUser.ID, teamUser.TeamID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(user)
}

func (h *Handler) ListInvites(c *fiber.Ctx) error {
	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	return c.JSON(h.service.ListInvites(c.Context(), teamUser.TeamID))
}
