package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
)

type teamResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type addUserResponse struct {
	TeamUser struct {
		ID   uint `json:"id"`
		User struct {
			ID          uint    `json:"id"`
			Email       string  `json:"email"`
			FirstName   *string `json:"first_name"`
			LastName    *string `json:"last_name"`
			IsSuperuser bool    `json:"is_superuser"`
		} `json:"user"`
		Role      string `json:"role"`
		SecretKey string `json:"secret_key"`
		CreatedAt string `json:"created_at"`
	} `json:"team_user"`
	Password *string `json:"password,omitempty"`
}

func TestCreateTeam_SuperuserSucceeds(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	admin := CreateTestUser(t, db, "admin-team@example.com", true)
	adminSess := CreateSessionForUser(t, db, admin)

	payload := `{"name":"Test Team"}`
	req := httptest.NewRequest("POST", "/api/v1/team/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	var tr teamResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if tr.Name != "Test Team" {
		t.Fatalf("expected team name 'Test Team', got %q", tr.Name)
	}
	if tr.Slug == "" {
		t.Fatalf("expected non-empty slug in response")
	}

	// Verify team exists in DB
	var team models.Team
	if err := db.Where("slug = ?", tr.Slug).First(&team).Error; err != nil {
		t.Fatalf("expected team to be persisted in DB: %v", err)
	}
}

func TestCreateTeam_NonSuperuserBlocked(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "user-team@example.com", false)
	userSess := CreateSessionForUser(t, db, user)

	payload := `{"name":"Forbidden Team"}`
	req := httptest.NewRequest("POST", "/api/v1/team/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(userSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d Forbidden for non-superuser, got %d", http.StatusForbidden, resp.StatusCode)
	}
}

func TestAddUser_AsAdminSucceeds(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create an existing user who will be the team admin (not superuser)
	owner := CreateTestUser(t, db, "owner@example.com", false)
	_, ownerTeamUser := CreateTeamAndTeamUser(t, db, "Owner Team", owner, "admin")

	// Create session for owner
	ownerSess := CreateSessionForUser(t, db, owner)

	// Add a new user to the team via API
	reqBody := map[string]interface{}{
		"email": "newuser@example.com",
		"role":  "member",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/team/add", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(ownerSess))
	req.Header.Set("X-Team-Slug", ownerTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK adding user as admin, got %d", resp.StatusCode)
	}

	var ar addUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		t.Fatalf("failed to decode add user response: %v", err)
	}

	if ar.TeamUser.User.Email != "newuser@example.com" {
		t.Fatalf("expected returned user email to be newuser@example.com, got %q", ar.TeamUser.User.Email)
	}

	// Verify the user exists in DB
	var user models.User
	if err := db.Where("email = ?", "newuser@example.com").First(&user).Error; err != nil {
		t.Fatalf("expected new user to be created in DB: %v", err)
	}

	// Verify team_user linkage exists
	var teamUser models.TeamUser
	if err := db.Where("user_id = ? AND team_id = ?", user.ID, ownerTeamUser.Team.ID).First(&teamUser).Error; err != nil {
		t.Fatalf("expected team_user to be created linking new user to team: %v", err)
	}
}

func TestAddUser_NonAdminForbidden(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create an existing user who will be a member (not admin)
	member := CreateTestUser(t, db, "member@example.com", false)
	_, memberTeamUser := CreateTeamAndTeamUser(t, db, "Member Team", member, "member")

	memberSess := CreateSessionForUser(t, db, member)

	reqBody := map[string]interface{}{
		"email": "another@example.com",
		"role":  "member",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/team/add", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(memberSess))
	req.Header.Set("X-Team-Slug", memberTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d Forbidden for non-admin, got %d", http.StatusForbidden, resp.StatusCode)
	}
}
