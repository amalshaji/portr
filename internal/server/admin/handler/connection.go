package handler

import (
	"strconv"

	"github.com/amalshaji/localport/internal/constants"
	"github.com/amalshaji/localport/internal/server/admin/service"
	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ListConnections(c *fiber.Ctx) error {
	connection_type := c.Query("type")

	pageNoStr := c.Query("pageNo")
	pageNo, err := strconv.Atoi(pageNoStr)
	if err != nil {
		pageNo = 1
	}

	var output any

	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	if connection_type == "active" {
		activeConnections := h.service.ListActiveConnections(c.Context(), teamUser.TeamID, int64(pageNo))
		count := h.service.GetActiveConnectionCount(c.Context(), teamUser.TeamID)
		output = PaginatedResponse[db.GetActiveConnectionsForTeamRow]{
			Data: activeConnections,
			Pagination: Pagination{
				Page:     pageNo,
				PageSize: service.DefaultPageSize,
				Total:    int(count),
			},
		}
	} else {
		recentConnections := h.service.ListRecentConnections(c.Context(), teamUser.TeamID, int64(pageNo))
		count := h.service.GetRecentConnectionCount(c.Context(), teamUser.TeamID)
		output = PaginatedResponse[db.GetRecentConnectionsForTeamRow]{
			Data: recentConnections,
			Pagination: Pagination{
				Page:     pageNo,
				PageSize: service.DefaultPageSize,
				Total:    int(count),
			},
		}
	}

	return c.JSON(output)
}

func (h *Handler) CreateConnection(c *fiber.Ctx) error {
	subdomain := c.Get("X-Subdomain")
	connectionType := c.Get("X-Connection-Type")

	var err error

	if connectionType == "" {
		return utils.ErrBadRequest(c, "connection type is required")
	}

	if connectionType == string(constants.Http) {
		if subdomain == "" {
			return utils.ErrBadRequest(c, "subdomain is required")
		}
	}

	secretKey := c.Get("X-SecretKey")
	if secretKey == "" {
		return utils.ErrBadRequest(c, "secret key is required")
	}

	var connection db.Connection

	if connectionType == string(constants.Http) {
		connection, err = h.service.RegisterNewHttpConnection(c.Context(), subdomain, secretKey)
	} else {
		connection, err = h.service.RegisterNewTcpConnection(c.Context(), secretKey)
	}

	if err != nil {
		return utils.ErrBadRequest(c, err.Error())
	}

	return c.JSON(fiber.Map{
		"connectionId": connection.ID,
	})
}
