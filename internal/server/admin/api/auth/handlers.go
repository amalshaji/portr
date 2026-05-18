package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/amalshaji/portr/internal/server/admin/services"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type githubOAuthService interface {
	IsEnabled() bool
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
	GetUser(ctx context.Context, token *oauth2.Token) (*services.GitHubUser, error)
}

type Handler struct {
	db            *gorm.DB
	store         *session.Store
	githubService githubOAuthService
	config        *serverConfig.AdminConfig
}

func NewHandler(db *gorm.DB, store *session.Store, cfg *serverConfig.AdminConfig) *Handler {
	var githubService githubOAuthService
	if service := services.NewGitHubService(cfg); service != nil {
		githubService = service
	}

	return &Handler{
		db:            db,
		store:         store,
		githubService: githubService,
		config:        cfg,
	}
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (h *Handler) GetAuthConfig(c *fiber.Ctx) error {
	var userCount int64
	h.db.Model(&models.User{}).Count(&userCount)

	githubEnabled := h.githubService != nil && h.githubService.IsEnabled()

	return c.JSON(fiber.Map{
		"is_first_signup":     userCount == 0,
		"github_auth_enabled": githubEnabled,
	})
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Check if this is the first user
	var userCount int64
	h.db.Model(&models.User{}).Count(&userCount)

	var user *models.User
	var err error

	if userCount == 0 {
		// Create first user as superuser
		user = &models.User{
			Email:       input.Email,
			IsSuperuser: true,
		}

		if err := user.SetPassword(input.Password); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create user",
			})
		}

		if err := h.db.Create(user).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"email": "Failed to create user",
			})
		}

		// Create default team
		team := &models.Team{
			Name: "Default Team",
		}
		if err := h.db.Create(team).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create team",
			})
		}

		// Add user to team
		teamUser := &models.TeamUser{
			UserID: user.ID,
			TeamID: team.ID,
			Role:   models.RoleAdmin,
		}
		if err := h.db.Create(teamUser).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to add user to team",
			})
		}

		log.Info("Created first superuser", "email", input.Email)
	} else {
		// Find existing user
		err = h.db.Where("email = ?", input.Email).First(&user).Error
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"email": "User does not exist",
			})
		}

		// Check password
		if !user.CheckPassword(input.Password) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"password": "Password is incorrect",
			})
		}
	}

	// Create session
	session := models.NewSession(user.ID)
	if err := h.db.Create(session).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "portr_session",
		Value:    session.Token,
		HTTPOnly: true,
		Secure:   true,
		MaxAge:   7 * 24 * 60 * 60,
		SameSite: "Lax",
	})

	// Get first team for redirect
	var team models.Team
	h.db.Joins("JOIN team_users ON team_users.team_id = team.id").
		Where("team_users.user_id = ?", user.ID).
		First(&team)

	return c.JSON(fiber.Map{
		"redirect_to": "/" + team.Slug + "/overview",
	})
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	token := c.Cookies("portr_session")
	if token != "" {
		h.db.Where("token = ?", token).Delete(&models.Session{})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "portr_session",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1,
	})

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) GitHubLogin(c *fiber.Ctx) error {
	if h.githubService == nil || !h.githubService.IsEnabled() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "GitHub authentication is not enabled",
		})
	}

	// Generate state parameter for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate state",
		})
	}

	// Store state in session for verification
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}
	sess.Set("oauth_state", state)
	if err := sess.Save(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save session",
		})
	}

	// Handle next URL parameter for post-login redirect
	nextURL := c.Query("next")
	if nextURL != "" {
		sess.Set("portr_next_url", nextURL)
		sess.Save()
	}

	authURL := h.githubService.GetAuthURL(state)

	// Return redirect response like Python implementation
	return c.Redirect(authURL, fiber.StatusFound)
}

func (h *Handler) GitHubCallback(c *fiber.Ctx) error {
	if h.githubService == nil || !h.githubService.IsEnabled() {
		return c.Redirect("/?code=github-disabled", fiber.StatusFound)
	}

	// Verify state parameter
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Redirect("/?code=invalid-session", fiber.StatusFound)
	}

	storedState := sess.Get("oauth_state")
	if storedState == nil || storedState != c.Query("state") {
		return c.Redirect("/?code=invalid-state", fiber.StatusFound)
	}

	// Clear the state from session
	sess.Delete("oauth_state")

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		return c.Redirect("/?code=no-code", fiber.StatusFound)
	}

	// Exchange code for token
	ctx := context.Background()
	token, err := h.githubService.ExchangeCode(ctx, code)
	if err != nil {
		log.Error("Failed to exchange GitHub code", "error", err)
		return c.Redirect("/?code=token-exchange-failed", fiber.StatusFound)
	}

	// Get user info from GitHub
	githubUser, err := h.githubService.GetUser(ctx, token)
	if err != nil {
		log.Error("Failed to get GitHub user", "error", err)
		return c.Redirect("/?code=user-fetch-failed", fiber.StatusFound)
	}

	if githubUser.Email == "" {
		return c.Redirect("/?code=private-email", fiber.StatusFound)
	}

	loginResult, errCode, err := h.resolveGitHubLogin(githubUser, token.AccessToken)
	if errCode != "" {
		return c.Redirect("/?code="+errCode, fiber.StatusFound)
	}
	if err != nil {
		log.Error("Database error during GitHub login", "error", err)
		return c.Redirect("/?code=database-error", fiber.StatusFound)
	}

	// Create session
	session := models.NewSession(loginResult.User.ID)
	if err := h.db.Create(session).Error; err != nil {
		return c.Redirect("/?code=session-creation-failed", fiber.StatusFound)
	}

	// Set authentication cookie (same as regular login)
	c.Cookie(&fiber.Cookie{
		Name:     "portr_session",
		Value:    session.Token,
		HTTPOnly: true,
		Secure:   true,
		MaxAge:   7 * 24 * 60 * 60,
		SameSite: "Lax",
	})

	// Get next URL or default redirect
	nextURL := sess.Get("portr_next_url")
	sess.Delete("portr_next_url")
	sess.Save()

	if loginResult.RedirectTeamSlug != "" {
		return c.Redirect("/"+loginResult.RedirectTeamSlug+"/overview", fiber.StatusFound)
	}

	if nextURL != nil {
		if nextURLStr, ok := nextURL.(string); ok && nextURLStr != "" {
			return c.Redirect(nextURLStr, fiber.StatusFound)
		}
	}

	// Get first team for redirect (same as regular login)
	var team models.Team
	h.db.Joins("JOIN team_users ON team_users.team_id = team.id").
		Where("team_users.user_id = ?", loginResult.User.ID).
		First(&team)

	if team.ID != 0 {
		return c.Redirect("/"+team.Slug+"/overview", fiber.StatusFound)
	}

	// Fallback redirect to root
	return c.Redirect("/", fiber.StatusFound)
}

type githubLoginResult struct {
	User             models.User
	RedirectTeamSlug string
}

func (h *Handler) resolveGitHubLogin(githubUser *services.GitHubUser, accessToken string) (*githubLoginResult, string, error) {
	var user models.User
	var githubUserRecord models.GithubUser

	err := h.db.Preload("User").Where("github_id = ?", githubUser.ID).First(&githubUserRecord).Error
	if err == nil {
		return &githubLoginResult{User: githubUserRecord.User}, "", nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, "", err
	}

	err = h.db.Where("email = ?", githubUser.Email).First(&user).Error
	if err == nil {
		githubUserRecord = models.GithubUser{
			GithubID:          githubUser.ID,
			GithubAccessToken: accessToken,
			GithubAvatarURL:   githubUser.AvatarURL,
			UserID:            user.ID,
		}

		if err := h.db.Create(&githubUserRecord).Error; err != nil {
			if updateErr := h.db.Where("user_id = ?", user.ID).Updates(&githubUserRecord).Error; updateErr != nil {
				return nil, "", updateErr
			}
		}

		return &githubLoginResult{User: user}, "", nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, "", err
	}

	return h.autoSignupGitHubUser(githubUser, accessToken)
}

func (h *Handler) autoSignupGitHubUser(githubUser *services.GitHubUser, accessToken string) (*githubLoginResult, string, error) {
	settings, err := models.GetOrCreateInstanceSettings(h.db)
	if err != nil {
		return nil, "", err
	}
	if !settings.AutoSignupEnabled {
		log.Warn("GitHub user attempted login but no account exists", "email", githubUser.Email)
		return nil, "auto-signup-disabled", nil
	}
	if !models.EmailMatchesAllowedDomains(githubUser.Email, settings.AutoSignupAllowedDomains) {
		log.Warn("GitHub auto signup rejected email domain", "email", githubUser.Email)
		return nil, "auto-signup-domain-denied", nil
	}
	if settings.AutoSignupTeamID == nil || *settings.AutoSignupTeamID == 0 {
		log.Error("GitHub auto signup is enabled without a target team")
		return nil, "auto-signup-team-missing", nil
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		return nil, "", tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var team models.Team
	if err := tx.First(&team, *settings.AutoSignupTeamID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return nil, "auto-signup-team-missing", nil
		}
		return nil, "", err
	}

	user := models.User{
		Email: strings.TrimSpace(githubUser.Email),
	}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	githubUserRecord := models.GithubUser{
		GithubID:          githubUser.ID,
		GithubAccessToken: accessToken,
		GithubAvatarURL:   githubUser.AvatarURL,
		UserID:            user.ID,
	}
	if err := tx.Create(&githubUserRecord).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	teamUser := models.TeamUser{
		UserID: user.ID,
		TeamID: team.ID,
		Role:   models.RoleMember,
	}
	if err := tx.Create(&teamUser).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, "", err
	}

	return &githubLoginResult{
		User:             user,
		RedirectTeamSlug: team.Slug,
	}, "", nil
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
