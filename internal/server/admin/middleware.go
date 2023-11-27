package admin

import (
	"slices"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/gofiber/fiber/v2"
)

var viewAuthMiddleware = func(c *fiber.Ctx) error {
	user := c.Locals("user").(*db.User)
	if user == nil {
		return c.Redirect("/")
	}
	return c.Next()
}

var apiAuthMiddleware = func(c *fiber.Ctx) error {
	user := c.Locals("user").(*db.User)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	return c.Next()
}

// Make sure to run these after running auth middlewares
var adminPermissionRequired = func(c *fiber.Ctx) error {
	user := c.Locals("user").(*db.User)
	if !slices.Contains([]db.UserRole{db.Admin, db.SuperUser}, user.Role) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "you need admin permissions to perform this action",
		})
	}
	return c.Next()
}
