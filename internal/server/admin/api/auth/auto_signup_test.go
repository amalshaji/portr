package auth

import (
	"context"
	"errors"
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
	user                  *services.GitHubUser
	state                 string
	verifiedEmail         string
	verifiedEmailErr      error
	verifiedEmailRequests int
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

func (f *fakeGitHubService) GetVerifiedEmail(ctx context.Context, token *oauth2.Token, publicEmail string) (string, error) {
	f.verifiedEmailRequests++
	if f.verifiedEmailErr != nil {
		return "", f.verifiedEmailErr
	}
	if f.verifiedEmail != "" {
		return f.verifiedEmail, nil
	}
	return "", nil
}

func TestGitHubCallbackAutoSignupCreatesUserAndTeamMembership(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	team := &models.Team{Name: "Engineering"}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	createAutoSignupSettings(t, db, models.AutoSignupDomain{Domain: "example.com", TeamID: team.ID})

	fakeService := &fakeGitHubService{
		verifiedEmail: "new-user@example.com",
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

func TestGitHubCallbackLinksExistingUserCaseInsensitively(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	team := &models.Team{Name: "Engineering"}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}
	existingUser := &models.User{Email: "Member@Example.com"}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(existingUser).Error; err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}
	if err := db.Create(&models.TeamUser{UserID: existingUser.ID, TeamID: team.ID, Role: models.RoleMember}).Error; err != nil {
		t.Fatalf("failed to create existing team membership: %v", err)
	}

	createAutoSignupSettings(t, db, models.AutoSignupDomain{Domain: "example.com", TeamID: team.ID})

	fakeService := &fakeGitHubService{
		verifiedEmail: "member@example.com",
		user: &services.GitHubUser{
			ID:    54321,
			Email: "member@example.com",
		},
	}
	app := newAuthTestApp(db, fakeService)

	resp := performGitHubCallback(t, app, fakeService)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected status 302 Found, got %d", resp.StatusCode)
	}

	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected GitHub login to reuse the existing user, got %d users", userCount)
	}

	var githubUser models.GithubUser
	if err := db.Where("github_id = ?", int64(54321)).First(&githubUser).Error; err != nil {
		t.Fatalf("expected GitHub user link to be created: %v", err)
	}
	if githubUser.UserID != existingUser.ID {
		t.Fatalf("expected GitHub account to link to user %d, got user %d", existingUser.ID, githubUser.UserID)
	}
}

func TestGitHubCallbackLinkedUserSkipsVerifiedEmailFetch(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	team := &models.Team{Name: "Engineering"}
	if err := db.Create(&team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}
	user := &models.User{Email: "member@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := db.Create(&models.TeamUser{UserID: user.ID, TeamID: team.ID, Role: models.RoleMember}).Error; err != nil {
		t.Fatalf("failed to create team membership: %v", err)
	}
	if err := db.Create(&models.GithubUser{GithubID: 98765, UserID: user.ID}).Error; err != nil {
		t.Fatalf("failed to create GitHub user link: %v", err)
	}

	fakeService := &fakeGitHubService{
		user:             &services.GitHubUser{ID: 98765},
		verifiedEmailErr: errors.New("github email API unavailable"),
	}
	app := newAuthTestApp(db, fakeService)

	resp := performGitHubCallback(t, app, fakeService)
	defer resp.Body.Close()

	if location := resp.Header.Get("Location"); location != "/engineering/overview" {
		t.Fatalf("expected linked user login to succeed, got redirect %q", location)
	}
	if fakeService.verifiedEmailRequests != 0 {
		t.Fatalf("expected linked login to skip verified email fetch, got %d requests", fakeService.verifiedEmailRequests)
	}
}

func TestGitHubCallbackAutoSignupUsesDomainTeamMapping(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	amalTeam := &models.Team{Name: "Amal"}
	if err := db.Create(amalTeam).Error; err != nil {
		t.Fatalf("failed to create amal team: %v", err)
	}
	engineeringTeam := &models.Team{Name: "Engineering"}
	if err := db.Create(engineeringTeam).Error; err != nil {
		t.Fatalf("failed to create engineering team: %v", err)
	}

	createAutoSignupSettings(t, db,
		models.AutoSignupDomain{Domain: "amal.sh", TeamID: amalTeam.ID},
		models.AutoSignupDomain{Domain: "example.com", TeamID: engineeringTeam.ID},
	)

	fakeService := &fakeGitHubService{
		verifiedEmail: "new-user@amal.sh",
		user: &services.GitHubUser{
			ID:        67890,
			Email:     "new-user@amal.sh",
			AvatarURL: "https://avatars.example.com/new-user",
		},
	}
	app := newAuthTestApp(db, fakeService)

	resp := performGitHubCallback(t, app, fakeService)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected status 302 Found, got %d", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); location != "/amal/overview" {
		t.Fatalf("expected redirect to amal team overview, got %q", location)
	}

	var user models.User
	if err := db.Where("email = ?", "new-user@amal.sh").First(&user).Error; err != nil {
		t.Fatalf("expected auto signup user to be created: %v", err)
	}

	var amalMembershipCount int64
	if err := db.Model(&models.TeamUser{}).Where("team_id = ? AND user_id = ?", amalTeam.ID, user.ID).Count(&amalMembershipCount).Error; err != nil {
		t.Fatalf("failed to count amal team memberships: %v", err)
	}
	if amalMembershipCount != 1 {
		t.Fatalf("expected user to be added to amal team, got %d memberships", amalMembershipCount)
	}

	var engineeringMembershipCount int64
	if err := db.Model(&models.TeamUser{}).Where("team_id = ? AND user_id = ?", engineeringTeam.ID, user.ID).Count(&engineeringMembershipCount).Error; err != nil {
		t.Fatalf("failed to count engineering team memberships: %v", err)
	}
	if engineeringMembershipCount != 0 {
		t.Fatalf("expected user not to be added to engineering team, got %d memberships", engineeringMembershipCount)
	}
}

func TestGitHubCallbackAutoSignupRejectsUntrustedDomain(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	team := &models.Team{Name: "Engineering"}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	createAutoSignupSettings(t, db, models.AutoSignupDomain{Domain: "example.com", TeamID: team.ID})

	fakeService := &fakeGitHubService{
		verifiedEmail: "new-user@other.example",
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

func TestGitHubCallbackAutoSignupRejectsUnverifiedEmail(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	team := &models.Team{Name: "Engineering"}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	createAutoSignupSettings(t, db, models.AutoSignupDomain{Domain: "example.com", TeamID: team.ID})

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
	if location := resp.Header.Get("Location"); location != "/?code=private-email" {
		t.Fatalf("expected private email redirect, got %q", location)
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no user to be created, got %d", count)
	}
}

func TestGitHubCallbackReportsVerifiedEmailFetchFailure(t *testing.T) {
	db, cleanup := newAuthTestDB(t)
	defer cleanup()

	fakeService := &fakeGitHubService{
		user:             &services.GitHubUser{ID: 12345},
		verifiedEmailErr: errors.New("github email API unavailable"),
	}
	app := newAuthTestApp(db, fakeService)

	resp := performGitHubCallback(t, app, fakeService)
	defer resp.Body.Close()

	if location := resp.Header.Get("Location"); location != "/?code=email-verification-failed" {
		t.Fatalf("expected verified email fetch failure redirect, got %q", location)
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

func createAutoSignupSettings(t *testing.T, db *gorm.DB, domains ...models.AutoSignupDomain) {
	t.Helper()

	settings := models.DefaultAutoSignupSettings()
	settings.AutoSignupEnabled = true
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("failed to create instance settings: %v", err)
	}
	for _, domain := range domains {
		if err := db.Create(&domain).Error; err != nil {
			t.Fatalf("failed to create auto signup domain %q: %v", domain.Domain, err)
		}
	}
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
		&models.AutoSignupSettings{},
		&models.AutoSignupDomain{},
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
