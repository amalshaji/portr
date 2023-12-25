package admin

import (
	"fmt"
	"slices"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/gofiber/fiber/v2"
)

var rootViewAuthMiddleware = func(c *fiber.Ctx) error {
	user := c.Locals("user").(*db.UserWithTeams)
	if user == nil {
		return c.Redirect("/")
	}
	return c.Next()
}

var teamViewAuthMiddleware = func(c *fiber.Ctx) error {
	user := c.Locals("user").(*db.UserWithTeams)
	if user == nil {
		return c.Redirect("/")
	}

	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	if teamUser == nil {
		return c.Redirect(fmt.Sprintf("/%s/overview", user.Teams[0].Slug))
	}

	return c.Next()
}

var apiAuthMiddleware = func(c *fiber.Ctx) error {
	user := c.Locals("user").(*db.UserWithTeams)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	return c.Next()
}

var apiTeamAuthMiddleware = func(c *fiber.Ctx) error {
	teamUser := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	if teamUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	return c.Next()
}

// Make sure to run these after running auth middlewares
var adminPermissionRequired = func(c *fiber.Ctx) error {
	user := c.Locals("teamUser").(*db.GetTeamMemberByUserIdAndTeamSlugRow)
	if !slices.Contains([]string{"admin", "superuser"}, user.Role) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "you need admin permissions to perform this action",
		})
	}
	return c.Next()
}

var superUserPermissionRequired = func(c *fiber.Ctx) error {
	user := c.Locals("user").(*db.UserWithTeams)
	if !user.IsSuperUser {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "you need superuser permissions to perform this action",
		})
	}
	return c.Next()
}
