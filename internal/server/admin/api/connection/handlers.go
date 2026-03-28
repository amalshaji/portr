package connection

import (
	"fmt"
	"time"

	"github.com/amalshaji/portr/internal/server/admin/middleware"
	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	store *session.Store
}

func NewHandler(db *gorm.DB, store *session.Store) *Handler {
	return &Handler{
		db:    db,
		store: store,
	}
}

type CreateConnectionInput struct {
	SecretKey      string  `json:"secret_key" validate:"required"`
	ConnectionType string  `json:"connection_type" validate:"required,oneof=http tcp"`
	Subdomain      *string `json:"subdomain"`
}

type ConnectionResponse struct {
	ID        string           `json:"id"`
	Type      string           `json:"type"`
	Subdomain *string          `json:"subdomain"`
	Port      *uint32          `json:"port"`
	Status    string           `json:"status"`
	CreatedAt string           `json:"created_at"`
	StartedAt *string          `json:"started_at"`
	ClosedAt  *string          `json:"closed_at"`
	CreatedBy TeamUserResponse `json:"created_by"`
	Team      TeamResponse     `json:"team"`
	Duration  *string          `json:"duration"`
}

type TeamUserResponse struct {
	ID   uint         `json:"id"`
	User UserResponse `json:"user"`
	Role string       `json:"role"`
}

type UserResponse struct {
	ID        uint    `json:"id"`
	Email     string  `json:"email"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

type TeamResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetConnections returns paginated list of connections for the team
func (h *Handler) GetConnections(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Parse query parameters
	queryType := c.Query("type", "recent") // active or recent
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 10)
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// Build query
	query := h.db.Model(&models.Connection{}).Where("team_id = ?", teamUser.TeamID)

	if queryType == "active" {
		query = query.Where("status = ?", models.ConnectionStatusActive)
	}

	var total int64
	query.Count(&total)

	var connections []models.Connection
	err := query.Preload("CreatedBy").Preload("CreatedBy.User").Preload("Team").
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&connections).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load connections",
		})
	}

	// Build response
	var items []ConnectionResponse
	for _, conn := range connections {
		item := ConnectionResponse{
			ID:        conn.ID,
			Type:      conn.Type,
			Subdomain: conn.Subdomain,
			Port:      conn.Port,
			Status:    conn.Status,
			CreatedAt: conn.CreatedAt.Format("2006-01-02T15:04:05Z"),
			CreatedBy: TeamUserResponse{
				ID: conn.CreatedBy.ID,
				User: UserResponse{
					ID:        conn.CreatedBy.User.ID,
					Email:     conn.CreatedBy.User.Email,
					FirstName: conn.CreatedBy.User.FirstName,
					LastName:  conn.CreatedBy.User.LastName,
				},
				Role: conn.CreatedBy.Role,
			},
			Team: TeamResponse{
				ID:   conn.Team.ID,
				Name: conn.Team.Name,
				Slug: conn.Team.Slug,
			},
		}

		if conn.StartedAt != nil {
			startedAtStr := conn.StartedAt.Format("2006-01-02T15:04:05Z")
			item.StartedAt = &startedAtStr
		}

		if conn.ClosedAt != nil {
			closedAtStr := conn.ClosedAt.Format("2006-01-02T15:04:05Z")
			item.ClosedAt = &closedAtStr
		}

		// Calculate duration if connection was started
		if duration := conn.Duration(); duration != nil {
			durationStr := formatDuration(*duration)
			item.Duration = &durationStr
		}

		items = append(items, item)
	}

	return c.JSON(fiber.Map{
		"count": total,
		"data":  items,
	})
}

// CreateConnection creates a new connection (used by tunnel client)
func (h *Handler) CreateConnection(c *fiber.Ctx) error {
	var input CreateConnectionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Find team user by secret key
	var teamUser models.TeamUser
	err := h.db.Preload("Team").Where("secret_key = ?", input.SecretKey).First(&teamUser).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid secret key",
		})
	}

	// Validate subdomain for HTTP connections
	if input.ConnectionType == models.ConnectionTypeHTTP {
		if input.Subdomain == nil || *input.Subdomain == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Subdomain is required for HTTP connections",
			})
		}

		// Check if subdomain is already in use
		var existingConn models.Connection
		err := h.db.Where("subdomain = ? AND status IN (?, ?)",
			*input.Subdomain,
			models.ConnectionStatusReserved,
			models.ConnectionStatusActive).First(&existingConn).Error

		if err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Subdomain already in use",
			})
		}
	}

	// Create connection
	connection := models.NewConnection(input.ConnectionType, input.Subdomain, &teamUser)

	if err := h.db.Create(connection).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create connection",
		})
	}

	return c.JSON(fiber.Map{
		"connection_id": connection.ID,
	})
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		return d.Round(time.Minute).String()
	}
	if d < 24*time.Hour {
		return d.Round(time.Minute).String()
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
