package handler

import (
	"net/http"

	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListSettings(c *fiber.Ctx) error {
	return c.JSON(h.service.ListSettings())
}

func (h *Handler) ListSettingsForSignupPage(c *fiber.Ctx) error {
	return c.JSON(h.service.ListSettingsForSignup())
}

func (h *Handler) UpdateEmailSettings(c *fiber.Ctx) error {
	var updatePayload service.UpdateEmailSettingsInput
	if err := c.BodyParser(&updatePayload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	user, err := h.service.UpdateEmailSettings(updatePayload)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(user)
}
