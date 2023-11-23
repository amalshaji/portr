package handler

import "github.com/gofiber/fiber/v2"

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	return c.JSON(h.service.ListUsers())
}

func (h *Handler) Me(c *fiber.Ctx) error {
	return c.JSON(c.Locals("user"))
}
