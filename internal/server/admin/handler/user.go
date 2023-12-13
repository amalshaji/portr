package handler

import (
	"net/http"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListTeamUsers(c *fiber.Ctx) error {
	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	return c.JSON(h.service.ListTeamUsers(c.Context(), teamUser.TeamID))
}

func (h *Handler) Me(c *fiber.Ctx) error {
	return c.JSON(c.Locals("user").(*db.UserWithTeams))
}

func (h *Handler) MeInTeam(c *fiber.Ctx) error {
	return c.JSON(c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow))
}

func (h *Handler) MeUpdate(c *fiber.Ctx) error {
	var updatePayload struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	if err := c.BodyParser(&updatePayload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}
	userFromLocals := c.Locals("user").(*db.UserWithTeams)
	result, err := h.service.UpdateUser(c.Context(), userFromLocals.ID, updatePayload.FirstName, updatePayload.LastName)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "failed to update profile info"})
	}
	return c.JSON(result)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	// expire all keys!
	c.ClearCookie()
	err := h.service.Logout(c.Context(), c.Cookies("localport-session"))
	if err != nil {
		h.log.Error("error while logging out", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "internal server error"})
	}
	return c.SendStatus(http.StatusOK)
}

func (h *Handler) RotateSecretKey(c *fiber.Ctx) error {
	result, err := h.service.RotateSecretKey(c.Context(), c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow).ID)
	if err != nil {
		h.log.Error("error while logging out", "error", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(result)
}
