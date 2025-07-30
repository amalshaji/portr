package team

import (
	"crypto/rand"
	"encoding/hex"

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

type NewTeamInput struct {
	Name string `json:"name" validate:"required"`
}

type AddUserInput struct {
	Email        string `json:"email" validate:"required,email"`
	Role         string `json:"role" validate:"required,oneof=admin member"`
	SetSuperuser bool   `json:"set_superuser"`
}

type TeamResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TeamUserForTeamResponse struct {
	ID        uint         `json:"id"`
	User      UserResponse `json:"user"`
	Role      string       `json:"role"`
	SecretKey string       `json:"secret_key"`
	CreatedAt string       `json:"created_at"`
}

type UserResponse struct {
	ID          uint                `json:"id"`
	Email       string              `json:"email"`
	FirstName   *string             `json:"first_name"`
	LastName    *string             `json:"last_name"`
	IsSuperuser bool                `json:"is_superuser"`
	GithubUser  *GithubUserResponse `json:"github_user,omitempty"`
}

type GithubUserResponse struct {
	GithubAvatarURL string `json:"github_avatar_url"`
}

type AddUserResponse struct {
	TeamUser *TeamUserForTeamResponse `json:"team_user"`
	Password *string                  `json:"password,omitempty"`
}

func (h *Handler) CreateTeam(c *fiber.Ctx) error {
	user := middleware.GetCurrentUser(c)
	if user == nil || !user.IsSuperuser {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Superuser access required",
		})
	}

	var input NewTeamInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Create team
	team := &models.Team{
		Name: input.Name,
	}

	if err := h.db.Create(team).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to create team",
		})
	}

	response := TeamResponse{
		ID:   team.ID,
		Name: team.Name,
		Slug: team.Slug,
	}

	return c.JSON(response)
}

func (h *Handler) GetTeamUsers(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Parse pagination parameters
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 10)
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var teamUsers []models.TeamUser
	var total int64

	// Get total count
	h.db.Model(&models.TeamUser{}).Where("team_id = ?", teamUser.TeamID).Count(&total)

	// Get paginated results
	err := h.db.Preload("User").Preload("User.GithubUser").
		Where("team_id = ?", teamUser.TeamID).
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&teamUsers).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load team users",
		})
	}

	// Build response
	var items []TeamUserForTeamResponse
	for _, tu := range teamUsers {
		userResp := UserResponse{
			ID:          tu.User.ID,
			Email:       tu.User.Email,
			FirstName:   tu.User.FirstName,
			LastName:    tu.User.LastName,
			IsSuperuser: tu.User.IsSuperuser,
		}

		if tu.User.GithubUser != nil {
			userResp.GithubUser = &GithubUserResponse{
				GithubAvatarURL: tu.User.GithubUser.GithubAvatarURL,
			}
		}

		items = append(items, TeamUserForTeamResponse{
			ID:        tu.ID,
			User:      userResp,
			Role:      tu.Role,
			SecretKey: tu.SecretKey,
			CreatedAt: tu.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return c.JSON(fiber.Map{
		"count": total,
		"data":  items,
	})
}

func (h *Handler) AddUser(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Check permissions
	if !teamUser.CanManageTeam() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var input AddUserInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Check if setting superuser is allowed
	if input.SetSuperuser && !teamUser.User.IsSuperuser {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only superuser can set superuser",
		})
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var user *models.User
	var password *string

	// Check if user already exists
	err := tx.Where("email = ?", input.Email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new user
			generatedPassword := generateRandomPassword()
			password = &generatedPassword

			user = &models.User{
				Email:       input.Email,
				IsSuperuser: input.SetSuperuser,
			}

			if err := user.SetPassword(generatedPassword); err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to hash password",
				})
			}

			if err := tx.Create(user).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Failed to create user",
				})
			}
		} else {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database error",
			})
		}
	} else {
		// User exists, update superuser status if needed
		if input.SetSuperuser && !user.IsSuperuser {
			user.IsSuperuser = true
			if err := tx.Save(user).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to update user",
				})
			}
		}
	}

	// Check if user is already in team
	var existingTeamUser models.TeamUser
	err = tx.Where("user_id = ? AND team_id = ?", user.ID, teamUser.TeamID).First(&existingTeamUser).Error
	if err == nil {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User is already in team",
		})
	}

	// Add user to team
	newTeamUser := &models.TeamUser{
		UserID: user.ID,
		TeamID: teamUser.TeamID,
		Role:   input.Role,
	}

	if err := tx.Create(newTeamUser).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add user to team",
		})
	}

	// Load full data for response
	if err := tx.Preload("User").Preload("User.GithubUser").First(newTeamUser, newTeamUser.ID).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load team user",
		})
	}

	tx.Commit()

	// Build response
	userResp := UserResponse{
		ID:          newTeamUser.User.ID,
		Email:       newTeamUser.User.Email,
		FirstName:   newTeamUser.User.FirstName,
		LastName:    newTeamUser.User.LastName,
		IsSuperuser: newTeamUser.User.IsSuperuser,
	}

	if newTeamUser.User.GithubUser != nil {
		userResp.GithubUser = &GithubUserResponse{
			GithubAvatarURL: newTeamUser.User.GithubUser.GithubAvatarURL,
		}
	}

	teamUserResp := &TeamUserForTeamResponse{
		ID:        newTeamUser.ID,
		User:      userResp,
		Role:      newTeamUser.Role,
		SecretKey: newTeamUser.SecretKey,
		CreatedAt: newTeamUser.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	response := AddUserResponse{
		TeamUser: teamUserResp,
		Password: password,
	}

	return c.JSON(response)
}

func (h *Handler) RemoveUser(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Check permissions
	if !teamUser.CanManageTeam() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	teamUserID, paramErr := c.ParamsInt("id")
	if paramErr != nil || teamUserID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid team user ID",
		})
	}

	// Find team user to delete
	var teamUserToDelete models.TeamUser
	err := h.db.Preload("User").Where("id = ? AND team_id = ?", teamUserID, teamUser.TeamID).First(&teamUserToDelete).Error
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found in team",
		})
	}

	// Check if trying to remove superuser (only superuser can do this)
	if teamUserToDelete.User.IsSuperuser && !teamUser.User.IsSuperuser {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only superuser can remove superuser from team",
		})
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete team user
	if err := tx.Delete(&teamUserToDelete).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove user from team",
		})
	}

	// Check if user has any other team memberships
	var otherTeamCount int64
	tx.Model(&models.TeamUser{}).Where("user_id = ?", teamUserToDelete.UserID).Count(&otherTeamCount)

	// If user has no other teams, delete the user entirely
	if otherTeamCount == 0 {
		if err := tx.Delete(&teamUserToDelete.User).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete user",
			})
		}
	}

	tx.Commit()

	return c.JSON(fiber.Map{"status": "ok"})
}

func generateRandomPassword() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)[:16]
}
