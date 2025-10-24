package team

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/amalshaji/portr/internal/server/admin/middleware"
	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/charmbracelet/log"
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

type ResetPasswordResponse struct {
	Password string `json:"password"`
}

func (h *Handler) CreateTeam(c *fiber.Ctx) error {
	user := middleware.GetCurrentUser(c)
	if user == nil || !user.IsSuperuser {
		log.Warn("CreateTeam: Unauthorized access attempt",
			"user_id", func() interface{} {
				if user != nil {
					return user.ID
				} else {
					return nil
				}
			}(),
			"is_superuser", user != nil && user.IsSuperuser)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Superuser access required",
		})
	}

	var input NewTeamInput
	if err := c.BodyParser(&input); err != nil {
		log.Error("CreateTeam: Invalid input", "user_id", user.ID, "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	log.Info("CreateTeam: Starting team creation",
		"user_id", user.ID,
		"user_email", user.Email,
		"team_name", input.Name)

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			log.Error("CreateTeam: Transaction panic recovered", "panic", r)
			tx.Rollback()
		}
	}()

	// Create team
	team := &models.Team{
		Name: input.Name,
	}

	if err := tx.Create(team).Error; err != nil {
		log.Error("CreateTeam: Failed to create team",
			"team_name", input.Name,
			"user_id", user.ID,
			"error", err)
		tx.Rollback()

		// Check for specific constraint violations
		if strings.Contains(err.Error(), "UNIQUE constraint failed: team.name") {
			log.Warn("CreateTeam: Team name conflict", "team_name", input.Name)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "A team with this name already exists, please choose a different name",
			})
		}

		if strings.Contains(err.Error(), "UNIQUE constraint failed: team.slug") {
			log.Warn("CreateTeam: Team slug conflict",
				"team_name", input.Name,
				"generated_slug", team.Slug)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "A team with this name already exists, please choose a different name",
			})
		}

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to create team",
		})
	}

	log.Info("CreateTeam: Successfully created team",
		"team_name", team.Name,
		"team_id", team.ID,
		"team_slug", team.Slug,
		"user_id", user.ID)

	// Add the creating user as an admin to the team
	teamUser := &models.TeamUser{
		UserID: user.ID,
		TeamID: team.ID,
		Role:   "admin",
	}

	if err := tx.Create(teamUser).Error; err != nil {
		log.Error("CreateTeam: Failed to add user as admin to team",
			"user_id", user.ID,
			"team_id", team.ID,
			"error", err)
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add user to team",
		})
	}

	log.Info("CreateTeam: Successfully added user as admin to team",
		"user_id", user.ID,
		"team_id", team.ID)

	tx.Commit()
	log.Info("CreateTeam: Transaction committed successfully",
		"team_name", team.Name,
		"team_id", team.ID)

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

	// Validate input
	if input.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email is required",
		})
	}

	// Basic email validation
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, regexErr := regexp.MatchString(emailRegex, input.Email)
	if regexErr != nil || !matched {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Please enter a valid email address",
		})
	}

	if input.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role is required",
		})
	}

	if input.Role != "admin" && input.Role != "member" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role must be either 'admin' or 'member'",
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
			// Create new user - always generate password for new users
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
		// User exists - check if they're part of any other teams
		var existingTeamCount int64
		tx.Model(&models.TeamUser{}).Where("user_id = ?", user.ID).Count(&existingTeamCount)

		// Only generate password if user is not part of any teams
		if existingTeamCount == 0 {
			generatedPassword := generateRandomPassword()
			password = &generatedPassword

			if err := user.SetPassword(generatedPassword); err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to hash password",
				})
			}
		}

		// Update superuser status if needed
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

func (h *Handler) ResetUserPassword(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Only superusers can reset passwords
	if !teamUser.User.IsSuperuser {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Superuser access required",
		})
	}

	teamUserID, paramErr := c.ParamsInt("id")
	if paramErr != nil || teamUserID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid team user ID",
		})
	}

	// Find team user to reset password for
	var targetTeamUser models.TeamUser
	err := h.db.Preload("User").Where("id = ? AND team_id = ?", teamUserID, teamUser.TeamID).First(&targetTeamUser).Error
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found in team",
		})
	}

	// Generate new password
	newPassword := generateRandomPassword()

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update user's password
	if err := targetTeamUser.User.SetPassword(newPassword); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	if err := tx.Save(&targetTeamUser.User).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user password",
		})
	}

	tx.Commit()

	return c.JSON(ResetPasswordResponse{
		Password: newPassword,
	})
}

func generateRandomPassword() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)[:16]
}
