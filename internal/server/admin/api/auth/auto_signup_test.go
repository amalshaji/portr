package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/amalshaji/portr/internal/server/admin/services"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeGitHubService struct {
	user  *services.GitHubUser
	state string
}

func (f *fakeGitHubService) IsEnabled() bool {
	return true
}

func (f *fakeGitHubService) GetAuthURL(state string) string {
	f.state = state
	return "/github/oauth?state=" + url.QueryEscape(state)
}

func (f *fakeGitHubService) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "github-token"}, nil
}

func (f *fakeGitHubService) GetUser(ctx context.Context, token *oauth2.Token) (*services.GitHubUser, error) {
	return f.user, nil
}

func TestGitHubCallbackAutoSignupCreatesUserAndTeamMembership(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	team := &models.Team{Name: "Engineering"}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	teamID := team.ID
	settings := models.DefaultInstanceSettings()
	settings.AutoSignupEnabled = true
	settings.AutoSignupAllowedDomains = "example.com"
	settings.AutoSignupTeamID = &teamID
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("failed to create instance settings: %v", err)
	}

	fakeService := &fakeGitHubService{
		user: &services.GitHubUser{
			ID:        12345,
			Email:     "new-user@example.com",
			AvatarURL: "https://avatars.example.com/new-user",
		},
	}
	app := newAuthTestApp(db, fakeService)

	resp := performGitHubCallback(t, app, fakeService)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected status 302 Found, got %d", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); location != "/engineering/overview" {
		t.Fatalf("expected redirect to auto signup team overview, got %q", location)
	}

	var user models.User
	if err := db.Where("email = ?", "new-user@example.com").First(&user).Error; err != nil {
		t.Fatalf("expected auto signup user to be created: %v", err)
	}

	var githubUser models.GithubUser
	if err := db.Where("github_id = ? AND user_id = ?", int64(12345), user.ID).First(&githubUser).Error; err != nil {
		t.Fatalf("expected github user link to be created: %v", err)
	}

	var teamUser models.TeamUser
	if err := db.Where("team_id = ? AND user_id = ?", team.ID, user.ID).First(&teamUser).Error; err != nil {
		t.Fatalf("expected team membership to be created: %v", err)
	}
	if teamUser.Role != models.RoleMember {
		t.Fatalf("expected auto signup team role %q, got %q", models.RoleMember, teamUser.Role)
	}
}

func TestGitHubCallbackAutoSignupRejectsUntrustedDomain(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	team := &models.Team{Name: "Engineering"}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	teamID := team.ID
	settings := models.DefaultInstanceSettings()
	settings.AutoSignupEnabled = true
	settings.AutoSignupAllowedDomains = "example.com"
	settings.AutoSignupTeamID = &teamID
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("failed to create instance settings: %v", err)
	}

	fakeService := &fakeGitHubService{
		user: &services.GitHubUser{
			ID:        12345,
			Email:     "new-user@other.example",
			AvatarURL: "https://avatars.example.com/new-user",
		},
	}
	app := newAuthTestApp(db, fakeService)

	resp := performGitHubCallback(t, app, fakeService)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected status 302 Found, got %d", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); location != "/?code=auto-signup-domain-denied" {
		t.Fatalf("expected domain denied redirect, got %q", location)
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no user to be created, got %d", count)
	}
}

func newAuthTestApp(db *gorm.DB, githubService githubOAuthService) *fiber.App {
	app := fiber.New()
	store := session.New()
	handler := &Handler{
		db:            db,
		store:         store,
		githubService: githubService,
		config: &serverConfig.AdminConfig{
			Domain:         "localhost:8000",
			Debug:          true,
			GithubClientID: "github-client",
			GithubSecret:   "github-secret",
		},
	}

	app.Get("/github", handler.GitHubLogin)
	app.Get("/github/callback", handler.GitHubCallback)

	return app
}

func performGitHubCallback(t *testing.T, app *fiber.App, fakeService *fakeGitHubService) *http.Response {
	t.Helper()

	loginReq := httptest.NewRequest("GET", "/github", nil)
	loginResp, err := app.Test(loginReq, -1)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusFound {
		t.Fatalf("expected login status 302 Found, got %d", loginResp.StatusCode)
	}
	cookies := loginResp.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected oauth state session cookie")
	}

	callbackReq := httptest.NewRequest("GET", "/github/callback?state="+url.QueryEscape(fakeService.state)+"&code=ok", nil)
	for _, cookie := range cookies {
		callbackReq.AddCookie(cookie)
	}

	callbackResp, err := app.Test(callbackReq, -1)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}

	return callbackResp
}

func newAuthTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test sqlite DB: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.GithubUser{},
		&models.Team{},
		&models.TeamUser{},
		&models.Session{},
		&models.InstanceSettings{},
	); err != nil {
		t.Fatalf("failed to auto migrate auth test models: %v", err)
	}

	cleanup := func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		_ = sqlDB.Close()
	}

	return db, cleanup
}
