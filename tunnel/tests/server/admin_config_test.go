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

func TestDownloadConfig_ValidSecretKeyReturnsConfig(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a user and team user to get a valid secret key
	user := CreateTestUser(t, db, "cfguser@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "Cfg Team", user, "admin")

	payload := map[string]string{
		"secret_key": teamUser.SecretKey,
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/config/download", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for valid download config, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	message, ok := body["message"].(string)
	if !ok || message == "" {
		t.Fatalf("expected 'message' string in response, got: %v", body)
	}

	// Basic containment checks: should include server_url, ssh_url and the secret key
	if !strings.Contains(message, "server_url:") || !strings.Contains(message, "ssh_url:") {
		t.Fatalf("expected config content to include server_url and ssh_url, got: %s", message)
	}
	if !strings.Contains(message, teamUser.SecretKey) {
		t.Fatalf("expected config content to include secret key, got: %s", message)
	}
}

func TestDownloadConfig_InvalidSecretKeyUnauthorized(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	payload := map[string]string{
		"secret_key": "this-does-not-exist",
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/config/download", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized for invalid secret key, got %d", resp.StatusCode)
	}
}

func TestGetSetupScript_WithTeamUser_ReturnsScript(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "scriptuser@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "Script Team", user, "admin")

	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/config/setup-script", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", teamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for setup-script, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	msg, ok := body["message"].(string)
	if !ok || msg == "" {
		t.Fatalf("expected 'message' string in response, got: %v", body)
	}

	// Should include the secret key and the server URL configured in test helper ("http://localhost:8001")
	if !strings.Contains(msg, teamUser.SecretKey) {
		t.Fatalf("expected setup script to contain secret key, got: %s", msg)
	}
	if !strings.Contains(msg, "http://localhost:8001") {
		t.Fatalf("expected setup script to contain server URL, got: %s", msg)
	}
}

func TestGetSetupScript_NoTeamContext_BadRequest(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "noteam@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/config/setup-script", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	// No X-Team-Slug header

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request when team context missing, got %d", resp.StatusCode)
	}
}

func TestGetStats_WithTeamUser_ReturnsTeamAndSystemStats(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create owner and another member for the team
	owner := CreateTestUser(t, db, "statsowner@example.com", false)
	_, ownerTeamUser := CreateTeamAndTeamUser(t, db, "Stats Team", owner, "admin")

	member := CreateTestUser(t, db, "statsmember@example.com", false)

	// Add the member to the same team as the owner directly to avoid creating a duplicate team
	teamUserForMember := &models.TeamUser{
		UserID: member.ID,
		TeamID: ownerTeamUser.TeamID,
		Role:   models.RoleMember,
	}
	if err := db.Create(teamUserForMember).Error; err != nil {
		t.Fatalf("failed to create additional team member: %v", err)
	}

	// Create an active connection for this team
	sub := "statssub"
	conn := models.NewConnection(models.ConnectionTypeHTTP, &sub, ownerTeamUser)
	conn.Status = models.ConnectionStatusActive
	if err := db.Create(conn).Error; err != nil {
		t.Fatalf("failed to create active connection: %v", err)
	}

	// session for owner
	sess := CreateSessionForUser(t, db, owner)

	req := httptest.NewRequest("GET", "/api/v1/config/stats", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", ownerTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for stats endpoint, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode stats response: %v", err)
	}

	teamStats, ok := body["team_stats"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'team_stats' object in response, got: %v", body)
	}

	activeConnections, ok := teamStats["active_connections"].(float64)
	if !ok {
		t.Fatalf("expected numeric 'active_connections' in team_stats, got: %v", teamStats)
	}
	if int(activeConnections) < 1 {
		t.Fatalf("expected at least 1 active connection, got %v", activeConnections)
	}

	teamMembers, ok := teamStats["team_members"].(float64)
	if !ok {
		t.Fatalf("expected numeric 'team_members' in team_stats, got: %v", teamStats)
	}
	if int(teamMembers) < 2 {
		t.Fatalf("expected at least 2 team members, got %v", teamMembers)
	}

	// Ensure system_stats exists
	if _, ok := body["system_stats"]; !ok {
		t.Fatalf("expected 'system_stats' in response, got: %v", body)
	}
}
