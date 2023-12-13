package handler

import (
	"github.com/amalshaji/localport/internal/server/db"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListConnections(c *fiber.Ctx) error {
	connection_type := c.Query("type")
	teamUser := c.Locals("teamUser").(*db.TeamUser)
	if connection_type == "active" {
		return c.JSON(h.service.ListActiveConnections(teamUser.TeamID))
	}
	return c.JSON(h.service.ListRecentConnections(teamUser.TeamID))
}
