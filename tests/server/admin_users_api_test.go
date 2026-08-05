package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverAdmin "github.com/amalshaji/portr/internal/server/admin"
	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/gorm"
)

type adminAddUserResponse struct {
	TeamUser struct {
		User struct {
			ID    uint   `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
		Role string `json:"role"`
	} `json:"team_user"`
	Team struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"team"`
	Password string `json:"password,omitempty"`
}

func TestAdminUsersAddDefaultsTeamAndReturnsPassword(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	admin := CreateTestUser(t, db, "admin@example.com", false)
	team, adminMembership := CreateTeamAndTeamUser(t, db, "Default Team", admin, models.RoleAdmin)
	if team.Slug != "default-team" {
		t.Fatalf("expected default team slug, got %q", team.Slug)
	}

	resp := addUserWithToken(t, srv, adminMembership.SecretKey, `{"email":"New.User@Example.com"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
	body := readBody(t, resp)
	if strings.Contains(body, "secret_key") || strings.Contains(body, adminMembership.SecretKey) {
		t.Fatalf("response leaked a secret key: %s", body)
	}

	var result adminAddUserResponse
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Team.Slug != "default-team" || result.TeamUser.Role != models.RoleMember {
		t.Fatalf("unexpected defaults: team=%q role=%q", result.Team.Slug, result.TeamUser.Role)
	}
	if result.Password == "" {
		t.Fatal("expected generated password")
	}
}

func TestAdminUsersAddOmitsPasswordWhenAutoSignupEnabled(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	admin := CreateTestUser(t, db, "admin@example.com", true)
	_, adminMembership := CreateTeamAndTeamUser(t, db, "Default Team", admin, models.RoleAdmin)
	settings := models.DefaultAutoSignupSettings()
	settings.ID = 1
	settings.AutoSignupEnabled = true
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("create auto signup settings: %v", err)
	}

	resp := addUserWithToken(t, srv, adminMembership.SecretKey, `{"email":"github@example.com","role":"admin"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
	body := readBody(t, resp)
	if strings.Contains(body, `"password"`) {
		t.Fatalf("expected password field to be omitted, got %s", body)
	}

	var user models.User
	if err := db.Where("email = ?", "github@example.com").First(&user).Error; err != nil {
		t.Fatalf("load created user: %v", err)
	}
	if user.Password != nil {
		t.Fatal("expected passwordless user")
	}
}

func TestAdminUsersAddAuthorization(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, db *gorm.DB, dbUser *models.User, dbTeam *models.Team) *models.TeamUser
		wantStatus int
	}{
		{
			name: "target team admin",
			setup: func(t *testing.T, db *gorm.DB, user *models.User, team *models.Team) *models.TeamUser {
				return createTeamMembership(t, db, user, team, models.RoleAdmin)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "target team member",
			setup: func(t *testing.T, db *gorm.DB, user *models.User, team *models.Team) *models.TeamUser {
				return createTeamMembership(t, db, user, team, models.RoleMember)
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := NewTestDB(t)
			defer cleanup()
			srv := NewTestServer(t, db)
			user := CreateTestUser(t, db, strings.ReplaceAll(tt.name, " ", "-")+"@example.com", false)
			team := models.Team{Name: "Target Team"}
			if err := db.Create(&team).Error; err != nil {
				t.Fatalf("create team: %v", err)
			}
			membership := tt.setup(t, db, user, &team)

			resp := addUserWithToken(t, srv, membership.SecretKey, `{"email":"invitee@example.com","team_slug":"target-team"}`)
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, resp.StatusCode, readBody(t, resp))
			}
		})
	}
}

func TestAdminUsersAddSuperuserCanTargetAnotherTeam(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	superuser := CreateTestUser(t, db, "root@example.com", true)
	_, credential := CreateTeamAndTeamUser(t, db, "Operators", superuser, models.RoleMember)
	target := models.Team{Name: "Engineering"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("create target team: %v", err)
	}

	resp := addUserWithToken(t, srv, credential.SecretKey, `{"email":"engineer@example.com","team_slug":"engineering"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
}

func TestAdminUsersAddCannotUseAnotherTeamsCredential(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "scoped-admin@example.com", false)
	_, sourceCredential := CreateTeamAndTeamUser(t, db, "Source Team", user, models.RoleMember)
	target := models.Team{Name: "Target Team"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("create target team: %v", err)
	}
	createTeamMembership(t, db, user, &target, models.RoleAdmin)

	resp := addUserWithToken(t, srv, sourceCredential.SecretKey, `{"email":"invitee@example.com","team_slug":"target-team"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for a credential scoped to another team, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
}

func TestAdminUsersAddRejectsInvalidOrUnauthorizedRequests(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)
	admin := CreateTestUser(t, db, "admin@example.com", false)
	_, credential := CreateTeamAndTeamUser(t, db, "Default Team", admin, models.RoleAdmin)

	tests := []struct {
		name       string
		token      string
		body       string
		wantStatus int
	}{
		{name: "missing token", body: `{"email":"new@example.com"}`, wantStatus: http.StatusUnauthorized},
		{name: "invalid token", token: "invalid", body: `{"email":"new@example.com"}`, wantStatus: http.StatusUnauthorized},
		{name: "invalid email", token: credential.SecretKey, body: `{"email":"not-an-email"}`, wantStatus: http.StatusBadRequest},
		{name: "invalid role", token: credential.SecretKey, body: `{"email":"new@example.com","role":"owner"}`, wantStatus: http.StatusBadRequest},
		{name: "missing team", token: credential.SecretKey, body: `{"email":"new@example.com","team_slug":"missing"}`, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := addUserWithToken(t, srv, tt.token, tt.body)
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, resp.StatusCode, readBody(t, resp))
			}
		})
	}
}

func TestAdminUsersAddReturnsConflictForExistingMembership(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)
	admin := CreateTestUser(t, db, "admin@example.com", false)
	team, credential := CreateTeamAndTeamUser(t, db, "Default Team", admin, models.RoleAdmin)
	existing := CreateTestUser(t, db, "existing@example.com", false)
	createTeamMembership(t, db, existing, team, models.RoleMember)

	resp := addUserWithToken(t, srv, credential.SecretKey, `{"email":"existing@example.com"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
}

func addUserWithToken(t *testing.T, srv *serverAdmin.Server, token, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := srv.App().Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func createTeamMembership(t *testing.T, db *gorm.DB, user *models.User, team *models.Team, role string) *models.TeamUser {
	t.Helper()
	membership := &models.TeamUser{UserID: user.ID, TeamID: team.ID, Role: role}
	if err := db.Create(membership).Error; err != nil {
		t.Fatalf("create team membership: %v", err)
	}
	if err := db.Preload("User").Preload("Team").First(membership, membership.ID).Error; err != nil {
		t.Fatalf("reload team membership: %v", err)
	}
	return membership
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return string(body)
}
