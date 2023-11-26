package handler

import (
	"net/http"

	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/server/db"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateInvite(c *fiber.Ctx) error {
	var payload service.CreateInviteInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	user, err := h.service.CreateInvite(payload, c.Locals("user").(db.User))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(user)
}

func (h *Handler) ListInvites(c *fiber.Ctx) error {
	return c.JSON(h.service.ListInvites())
}
