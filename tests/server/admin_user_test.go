package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
)

func TestGetCurrentUser_WithTeamContext_ReturnsUser(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "meuser@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "Me Team", user, "admin")

	// create session for user
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", teamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for GET /user/me, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	// Validate returned fields
	userObj, ok := body["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'user' object in response, got: %v", body)
	}
	if userObj["email"] != user.Email {
		t.Fatalf("expected email %s, got %v", user.Email, userObj["email"])
	}

	teamObj, ok := body["team"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'team' object in response, got: %v", body)
	}
	if teamObj["slug"] != teamUser.Team.Slug {
		t.Fatalf("expected team slug %s, got %v", teamUser.Team.Slug, teamObj["slug"])
	}
}

func TestGetUserTeams_ReturnsAllTeams(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "multiteam@example.com", false)

	// Create two teams and add same user to both
	teamA, _ := CreateTeamAndTeamUser(t, db, "Team A", user, "admin")
	teamB, _ := CreateTeamAndTeamUser(t, db, "Team B", user, "member")

	// Create session
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/user/me/teams", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for GET /user/me/teams, got %d", resp.StatusCode)
	}

	var teams []models.Team
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		t.Fatalf("failed to decode teams response: %v", err)
	}

	// Check that both team slugs are present
	foundA, foundB := false, false
	for _, tm := range teams {
		if tm.Slug == teamA.Slug {
			foundA = true
		}
		if tm.Slug == teamB.Slug {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("expected both team slugs %q and %q in response, got %+v", teamA.Slug, teamB.Slug, teams)
	}
}

func TestUpdateUser_UpdatesFields(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "updateuser@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	payload := map[string]interface{}{
		"first_name": "Alice",
		"last_name":  "Smith",
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/user/me/update", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(sess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for PATCH /user/me/update, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["first_name"] != "Alice" || body["last_name"] != "Smith" {
		t.Fatalf("expected updated names in response, got: %v", body)
	}

	// Verify persisted in DB
	var persisted models.User
	if err := db.First(&persisted, user.ID).Error; err != nil {
		t.Fatalf("failed to reload user from DB: %v", err)
	}
	if persisted.FirstName == nil || *persisted.FirstName != "Alice" {
		t.Fatalf("expected persisted first_name 'Alice', got %v", persisted.FirstName)
	}
	if persisted.LastName == nil || *persisted.LastName != "Smith" {
		t.Fatalf("expected persisted last_name 'Smith', got %v", persisted.LastName)
	}
}

func TestChangePassword_SucceedsAndPersists(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "changepass@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	payload := map[string]interface{}{
		"password": "newpassword123",
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/api/v1/user/me/change-password", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(sess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for password change, got %d", resp.StatusCode)
	}

	// Verify password was updated in DB by reloading user and checking password
	var persisted models.User
	if err := db.First(&persisted, user.ID).Error; err != nil {
		t.Fatalf("failed to reload user from DB: %v", err)
	}

	if !persisted.CheckPassword("newpassword123") {
		t.Fatalf("expected password to be updated and check to succeed")
	}
}

func TestRotateSecretKey_SucceedsForTeamUser(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "rotatesecret@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "Rotate Team", user, "admin")

	sess := CreateSessionForUser(t, db, user)

	// store original key
	origKey := teamUser.SecretKey

	req := httptest.NewRequest("PATCH", "/api/v1/user/me/rotate-secret-key", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", teamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for rotate secret key, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	newKey, ok := body["secret_key"].(string)
	if !ok || newKey == "" {
		t.Fatalf("expected 'secret_key' in response, got: %v", body)
	}
	if newKey == origKey {
		t.Fatalf("expected secret key to change, but it did not")
	}

	// Verify DB updated
	var refreshed models.TeamUser
	if err := db.First(&refreshed, teamUser.ID).Error; err != nil {
		t.Fatalf("failed to reload team user from DB: %v", err)
	}
	if refreshed.SecretKey == origKey {
		t.Fatalf("expected persisted secret key to differ from original")
	}
}
