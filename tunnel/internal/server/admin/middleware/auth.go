package middleware

import (
	"fmt"
	"time"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AuthMiddleware struct {
	db *gorm.DB
}

func NewAuthMiddleware(db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{
		db: db,
	}
}

func (m *AuthMiddleware) RequireAuth(c *fiber.Ctx) error {
	if err := m.checkAuth(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return c.Next()
}

func (m *AuthMiddleware) RequireAuthRedirect(c *fiber.Ctx) error {
	err := m.checkAuth(c)
	if err != nil {
		if c.Path() != "/" {
			return c.Redirect("/", fiber.StatusFound)
		}
		return c.Next()
	}

	// Handle no error
	if c.Path() == "/" {
		// Get user from locals
		user := GetCurrentUser(c)
		if user == nil {
			return c.Next()
		}

		// Find user's first team
		var teamUser models.TeamUser
		err := m.db.Preload("Team").
			Where("user_id = ?", user.ID).
			First(&teamUser).Error

		if err == nil && teamUser.Team.Slug != "" {
			return c.Redirect("/"+teamUser.Team.Slug+"/overview", fiber.StatusFound)
		}
	}

	return c.Next()
}

func (m *AuthMiddleware) checkAuth(c *fiber.Ctx) error {
	token := c.Cookies("portr_session")
	if token == "" {
		return fmt.Errorf("no session token provided")
	}

	var session models.Session
	if err := m.db.Preload("User").Where("token = ? AND expires_at > ?", token, time.Now()).First(&session).Error; err != nil {
		return fmt.Errorf("invalid or expired session: %w", err)
	}

	c.Locals("user", &session.User)
	return nil
}

func (m *AuthMiddleware) RequireTeamUser(c *fiber.Ctx) error {
	// First check if user is authenticated
	if err := m.checkAuth(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	teamSlug := c.Get("X-Team-Slug")

	if teamSlug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team slug required",
		})
	}

	// Load team user
	var teamUser models.TeamUser
	err := m.db.Preload("Team").Preload("User").
		Joins("JOIN team ON team.id = team_users.team_id").
		Where("team.slug = ? AND team_users.user_id = ?", teamSlug, user.ID).
		First(&teamUser).Error

	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Store team user in context
	c.Locals("team_user", &teamUser)
	return c.Next()
}

func (m *AuthMiddleware) RequireAdmin(c *fiber.Ctx) error {
	// First check authentication
	if err := m.checkAuth(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	teamSlug := c.Get("X-Team-Slug")
	if teamSlug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team slug required",
		})
	}

	// Load team user
	var teamUser models.TeamUser
	err := m.db.Preload("Team").Preload("User").
		Joins("JOIN team ON team.id = team_users.team_id").
		Where("team.slug = ? AND team_users.user_id = ?", teamSlug, user.ID).
		First(&teamUser).Error

	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Check admin permissions
	if !teamUser.IsAdmin() && !user.IsSuperuser {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Store team user in context
	c.Locals("team_user", &teamUser)
	return c.Next()
}

func (m *AuthMiddleware) RequireSuperuser(c *fiber.Ctx) error {
	// First check if user is authenticated
	if err := m.checkAuth(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if !user.IsSuperuser {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Superuser access required",
		})
	}

	return c.Next()
}

func GetCurrentUser(c *fiber.Ctx) *models.User {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return nil
	}
	return user
}

func GetCurrentTeamUser(c *fiber.Ctx) *models.TeamUser {
	teamUser, ok := c.Locals("team_user").(*models.TeamUser)
	if !ok {
		return nil
	}
	return teamUser
}
