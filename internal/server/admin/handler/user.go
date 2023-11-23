package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	return c.JSON(h.service.ListUsers())
}

func (h *Handler) Me(c *fiber.Ctx) error {
	return c.JSON(c.Locals("user"))
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	// expire all keys!
	c.ClearCookie()
	// TODO: delete the session from db as well
	err := h.service.Logout(c.Cookies("localport-session"))
	if err != nil {
		h.log.Error("error while logging out", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "internal server error"})
	}
	return c.SendStatus(http.StatusOK)
}
