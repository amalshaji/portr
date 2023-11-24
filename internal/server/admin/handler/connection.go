package handler

import "github.com/gofiber/fiber/v2"

func (h *Handler) ListConnections(c *fiber.Ctx) error {
	connection_type := c.Query("type")
	if connection_type == "active" {
		return c.JSON(h.service.ListActiveConnections())
	}
	return c.JSON(h.service.ListRecentConnections())
}
