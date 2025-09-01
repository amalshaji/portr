package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	serverAdmin "github.com/amalshaji/portr/internal/server/admin"
	"github.com/amalshaji/portr/internal/server/admin/models"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewTestDB returns an in-memory sqlite gorm.DB pre-migrated with admin models,
// and a cleanup function to close the underlying sql.DB.
func NewTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	// Use shared in-memory DB so multiple connections (if any) can see the same DB.
	dialector := sqlite.Open("file::memory:?cache=shared")
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test sqlite DB: %v", err)
	}

	// Enable SQLite pragmas for better performance and concurrency in tests
	err = db.Exec("PRAGMA journal_mode=WAL").Error
	if err != nil {
		t.Fatalf("failed to set journal_mode: %v", err)
	}

	err = db.Exec("PRAGMA busy_timeout=5000").Error
	if err != nil {
		t.Fatalf("failed to set busy_timeout: %v", err)
	}

	err = db.Exec("PRAGMA cache_size=10000").Error
	if err != nil {
		t.Fatalf("failed to set cache_size: %v", err)
	}

	// Run AutoMigrate for admin models used by tests.
	if err := db.AutoMigrate(
		&models.User{},
		&models.GithubUser{},
		&models.Team{},
		&models.TeamUser{},
		&models.Session{},
		&models.Connection{},
	); err != nil {
		t.Fatalf("failed to auto migrate admin models: %v", err)
	}

	cleanup := func() {
		sqlDB, err := db.DB()
		if err != nil {
			// Nothing to do if we cannot get underlying DB
			return
		}
		_ = sqlDB.Close()
	}

	return db, cleanup
}

// NewTestServer creates an admin.Server configured for tests using the provided DB.
func NewTestServer(t *testing.T, db *gorm.DB) *serverAdmin.Server {
	t.Helper()

	cfg := &serverConfig.AdminConfig{
		Port:           0,
		Domain:         "localhost:8000",
		Debug:          true,
		UseVite:        false,
		GithubClientID: "",
		GithubSecret:   "",
		ServerURL:      "http://localhost:8001",
		SshURL:         "localhost:2222",
	}

	srv := serverAdmin.NewServer(cfg, db)
	return srv
}

// CreateTestUser creates a user record in the DB with the given email and superuser flag.
// The function sets a deterministic password (useful for tests) but returns the user object.
func CreateTestUser(t *testing.T, db *gorm.DB, email string, isSuperuser bool) *models.User {
	t.Helper()

	user := &models.User{
		Email:       email,
		IsSuperuser: isSuperuser,
	}

	// Use a fixed password for tests
	if err := user.SetPassword("password123"); err != nil {
		t.Fatalf("failed to set password on test user: %v", err)
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Reload to ensure defaults/timestamps are populated
	if err := db.First(user, user.ID).Error; err != nil {
		t.Fatalf("failed to reload created user: %v", err)
	}

	return user
}

// CreateSessionForUser creates a session row for the provided user and returns it.
func CreateSessionForUser(t *testing.T, db *gorm.DB, user *models.User) *models.Session {
	t.Helper()

	session := models.NewSession(user.ID)
	// Make expiration deterministic (optional)
	session.ExpiresAt = time.Now().Add(24 * time.Hour)

	if err := db.Create(session).Error; err != nil {
		t.Fatalf("failed to create session for user %d: %v", user.ID, err)
	}

	// reload
	if err := db.First(session, session.ID).Error; err != nil {
		t.Fatalf("failed to reload session: %v", err)
	}

	return session
}

// SessionCookieValue returns the cookie header value to authenticate requests with the session.
func SessionCookieValue(s *models.Session) string {
	return fmt.Sprintf("portr_session=%s", s.Token)
}

// CreateTeamAndTeamUser creates a team and adds the provided user as a TeamUser with the given role.
// Returns the created team and teamUser.
func CreateTeamAndTeamUser(t *testing.T, db *gorm.DB, teamName string, user *models.User, role string) (*models.Team, *models.TeamUser) {
	t.Helper()

	team := &models.Team{
		Name: teamName,
	}

	if err := db.Create(team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	teamUser := &models.TeamUser{
		UserID: user.ID,
		TeamID: team.ID,
		Role:   role,
	}

	if err := db.Create(teamUser).Error; err != nil {
		t.Fatalf("failed to create team user: %v", err)
	}

	// preload associations
	if err := db.Preload("Team").Preload("User").First(teamUser, teamUser.ID).Error; err != nil {
		t.Fatalf("failed to reload team user: %v", err)
	}

	return team, teamUser
}

// DoRequest executes an http.Request against the provided server and returns the response.
// It fails the test on error.
func DoRequest(t *testing.T, srv *serverAdmin.Server, req *http.Request) *http.Response {
	t.Helper()

	resp, err := srv.App().Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}
