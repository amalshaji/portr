package handler

import (
	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListConnections(c *fiber.Ctx) error {
	connection_type := c.Query("type")
	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	if connection_type == "active" {
		return c.JSON(h.service.ListActiveConnections(c.Context(), teamUser.TeamID))
	}
	return c.JSON(h.service.ListRecentConnections(c.Context(), teamUser.TeamID))
}
