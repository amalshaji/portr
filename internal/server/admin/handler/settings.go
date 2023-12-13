package handler

import (
	"net/http"

	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListSettings(c *fiber.Ctx) error {
	return c.JSON(h.service.ListSettings(c.Context()))
}

func (h *Handler) UpdateEmailSettings(c *fiber.Ctx) error {
	var updatePayload service.UpdateEmailSettingsInput
	if err := c.BodyParser(&updatePayload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	result, err := h.service.UpdateEmailSettings(c.Context(), updatePayload)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(result)
}
