package handler

import "github.com/gofiber/fiber/v2"

func (h *Handler) ListActiveConnections(c *fiber.Ctx) error {
	return c.JSON(h.service.ListActiveConnections())
}
