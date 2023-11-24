package handler

import "github.com/gofiber/fiber/v2"

func (h *Handler) ListSettings(c *fiber.Ctx) error {
	return c.JSON(h.service.ListSettings())
}

func (h *Handler) ListSettingsForSignupPage(c *fiber.Ctx) error {
	return c.JSON(h.service.ListSettingsForSignup())
}
