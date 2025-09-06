package user

import (
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

type UserUpdateInput struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

type ChangePasswordInput struct {
	Password string `json:"password" validate:"required,min=8"`
}

type TeamUserResponse struct {
	ID        uint         `json:"id"`
	User      UserResponse `json:"user"`
	Team      TeamResponse `json:"team"`
	Role      string       `json:"role"`
	SecretKey string       `json:"secret_key"`
	CreatedAt string       `json:"created_at"`
}

type UserResponse struct {
	ID          uint                 `json:"id"`
	Email       string               `json:"email"`
	FirstName   *string              `json:"first_name"`
	LastName    *string              `json:"last_name"`
	IsSuperuser bool                 `json:"is_superuser"`
	GithubUser  *GithubUserResponse  `json:"github_user,omitempty"`
}

type GithubUserResponse struct {
	GithubAvatarURL string `json:"github_avatar_url"`
}

type TeamResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetCurrentUser returns current user info with team context
func (h *Handler) GetCurrentUser(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Load full user and team data including GitHub user
	if err := h.db.Preload("User").Preload("User.GithubUser").Preload("Team").First(teamUser, teamUser.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load user data",
		})
	}

	// Build user response with optional GitHub data
	userResponse := UserResponse{
		ID:          teamUser.User.ID,
		Email:       teamUser.User.Email,
		FirstName:   teamUser.User.FirstName,
		LastName:    teamUser.User.LastName,
		IsSuperuser: teamUser.User.IsSuperuser,
	}

	// Add GitHub user data if it exists
	if teamUser.User.GithubUser != nil {
		userResponse.GithubUser = &GithubUserResponse{
			GithubAvatarURL: teamUser.User.GithubUser.GithubAvatarURL,
		}
	}

	response := TeamUserResponse{
		ID:   teamUser.ID,
		User: userResponse,
		Team: TeamResponse{
			ID:   teamUser.Team.ID,
			Name: teamUser.Team.Name,
			Slug: teamUser.Team.Slug,
		},
		Role:      teamUser.Role,
		SecretKey: teamUser.SecretKey,
		CreatedAt: teamUser.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return c.JSON(response)
}

// GetUserTeams returns all teams the current user belongs to
func (h *Handler) GetUserTeams(c *fiber.Ctx) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var teams []models.Team
	err := h.db.Joins("JOIN team_users ON team_users.team_id = team.id").
		Where("team_users.user_id = ?", user.ID).
		Find(&teams).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load teams",
		})
	}

	var response []TeamResponse
	for _, team := range teams {
		response = append(response, TeamResponse{
			ID:   team.ID,
			Name: team.Name,
			Slug: team.Slug,
		})
	}

	return c.JSON(response)
}

// UpdateUser updates the current user's profile
func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var input UserUpdateInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Update user fields
	if input.FirstName != nil {
		user.FirstName = input.FirstName
	}
	if input.LastName != nil {
		user.LastName = input.LastName
	}

	if err := h.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	response := UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		IsSuperuser: user.IsSuperuser,
	}

	return c.JSON(response)
}

// ChangePassword changes the current user's password
func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var input ChangePasswordInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Set new password
	if err := user.SetPassword(input.Password); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	if err := h.db.Save(user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update password",
		})
	}

	response := UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		IsSuperuser: user.IsSuperuser,
	}

	return c.JSON(response)
}

// RotateSecretKey generates a new secret key for the current team user
func (h *Handler) RotateSecretKey(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Generate new secret key
	teamUser.SecretKey = models.GenerateSecretKey()

	if err := h.db.Save(teamUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to rotate secret key",
		})
	}

	return c.JSON(fiber.Map{
		"secret_key": teamUser.SecretKey,
	})
}
