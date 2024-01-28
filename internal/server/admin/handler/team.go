package handler

import (
	"github.com/amalshaji/portr/internal/server/admin/service"
	db "github.com/amalshaji/portr/internal/server/db/models"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateTeam(c *fiber.Ctx) error {
	var createTeamInput service.CreateTeamInput
	if err := utils.BodyParser(c, &createTeamInput); err != nil {
		return utils.ErrBadRequest(c, err.Error())
	}
	user := c.Locals("user").(*db.UserWithTeams)
	team, err := h.service.CreateFirstTeam(c.Context(), createTeamInput, user.ID)
	if err != nil {
		return utils.ErrInternalServerError(c, err.Error())
	}
	return c.JSON(team)
}

func (h *Handler) AddMember(c *fiber.Ctx) error {
	var payload service.AddMemberInput
	if err := utils.BodyParser(c, &payload); err != nil {
		return utils.ErrBadRequest(c, err)
	}
	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	result, err := h.service.AddMember(c.Context(), payload, teamUser.TeamID, teamUser.ID)
	if err != nil {
		h.log.Error("error while logging out", "error", err)
		return utils.ErrInternalServerError(c, "internal server error")
	}
	return c.JSON(result)
}
