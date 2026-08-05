package team

import (
	"errors"
	"strings"

	"github.com/amalshaji/portr/internal/server/admin/middleware"
	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/amalshaji/portr/internal/server/admin/services"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	store     *session.Store
	teamUsers *services.TeamUserService
}

func NewHandler(db *gorm.DB, store *session.Store) *Handler {
	return &Handler{
		db:        db,
		store:     store,
		teamUsers: services.NewTeamUserService(db),
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

type AdminAddUserInput struct {
	Email    string `json:"email"`
	TeamSlug string `json:"team_slug"`
	Role     string `json:"role"`
}

type TeamResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TeamUserListResponse struct {
	ID        uint         `json:"id"`
	User      UserResponse `json:"user"`
	Role      string       `json:"role"`
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
	TeamUser TeamUserListResponse `json:"team_user"`
	Team     TeamResponse         `json:"team"`
	Password string               `json:"password,omitempty"`
}

type ResetPasswordResponse struct {
	Password string `json:"password"`
}

func (h *Handler) ListTeams(c *fiber.Ctx) error {
	var teams []models.Team
	if err := h.db.Order("name ASC").Find(&teams).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load teams",
		})
	}

	response := make([]TeamResponse, 0, len(teams))
	for _, team := range teams {
		response = append(response, TeamResponse{
			ID:   team.ID,
			Name: team.Name,
			Slug: team.Slug,
		})
	}

	return c.JSON(response)
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
	if page < 1 {
		page = 1
	}
	pageSize := c.QueryInt("page_size", 10)
	if pageSize < 1 {
		pageSize = 10
	}
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

	var items []TeamUserListResponse
	for _, tu := range teamUsers {
		items = append(items, teamUserListResponseFor(tu))
	}

	return c.JSON(fiber.Map{
		"count": total,
		"data":  items,
	})
}

func (h *Handler) AddUser(c *fiber.Ctx) error {
	actor := middleware.GetCurrentTeamUser(c)
	if actor == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	var input AddUserInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}
	return h.inviteUser(c, actor.ID, services.InviteUserInput{
		Email:        input.Email,
		TeamSlug:     actor.Team.Slug,
		Role:         input.Role,
		SetSuperuser: input.SetSuperuser,
	})
}

func (h *Handler) AddUserWithAPIKey(c *fiber.Ctx) error {
	actor := middleware.GetCurrentTeamUser(c)
	if actor == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var input AdminAddUserInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	return h.inviteUser(c, actor.ID, services.InviteUserInput{
		Email:    input.Email,
		TeamSlug: input.TeamSlug,
		Role:     input.Role,
	})
}

func (h *Handler) inviteUser(c *fiber.Ctx, actorTeamUserID uint, input services.InviteUserInput) error {
	result, err := h.teamUsers.Invite(c.UserContext(), actorTeamUserID, input)
	if err != nil {
		return inviteUserError(c, err)
	}

	teamUser := teamUserListResponseFor(result.TeamUser)
	return c.JSON(AddUserResponse{
		TeamUser: teamUser,
		Team: TeamResponse{
			ID:   result.TeamUser.Team.ID,
			Name: result.TeamUser.Team.Name,
			Slug: result.TeamUser.Team.Slug,
		},
		Password: result.Password,
	})
}

func inviteUserError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidEmail):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Please enter a valid email address"})
	case errors.Is(err, services.ErrInvalidRole):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role must be either 'admin' or 'member'"})
	case errors.Is(err, services.ErrTeamNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Team not found"})
	case errors.Is(err, services.ErrAdminAccessRequired):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Admin access required"})
	case errors.Is(err, services.ErrSuperuserAccessRequired):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only superuser can set superuser"})
	case errors.Is(err, services.ErrUserAlreadyInTeam):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "User is already in team"})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to add user to team"})
	}
}

func teamUserListResponseFor(teamUser models.TeamUser) TeamUserListResponse {
	user := UserResponse{
		ID:          teamUser.User.ID,
		Email:       teamUser.User.Email,
		FirstName:   teamUser.User.FirstName,
		LastName:    teamUser.User.LastName,
		IsSuperuser: teamUser.User.IsSuperuser,
	}
	if teamUser.User.GithubUser != nil {
		user.GithubUser = &GithubUserResponse{GithubAvatarURL: teamUser.User.GithubUser.GithubAvatarURL}
	}

	return TeamUserListResponse{
		ID:        teamUser.ID,
		User:      user,
		Role:      teamUser.Role,
		CreatedAt: teamUser.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
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

	newPassword, err := services.GenerateRandomPassword()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate password",
		})
	}

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
