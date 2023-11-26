package handler

import (
	"net/http"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	return c.JSON(h.service.ListUsers())
}

func (h *Handler) Me(c *fiber.Ctx) error {
	return c.JSON(c.Locals("user"))
}

func (h *Handler) MeUpdate(c *fiber.Ctx) error {
	var updatePayload struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	if err := c.BodyParser(&updatePayload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	userFromLocals := c.Locals("user").(db.User)
	user, err := h.service.UpdateUser(&userFromLocals, updatePayload.FirstName, updatePayload.LastName)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "failed to update profile info"})
	}
	return c.JSON(user)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	// expire all keys!
	c.ClearCookie()
	err := h.service.Logout(c.Cookies("localport-session"))
	if err != nil {
		h.log.Error("error while logging out", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "internal server error"})
	}
	return c.SendStatus(http.StatusOK)
}

func (h *Handler) RotateSecretKey(c *fiber.Ctx) error {
	user, err := h.service.RotateSecretKey(c.Locals("user").(db.User))
	if err != nil {
		h.log.Error("error while logging out", "error", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(user)
}
