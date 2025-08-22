package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
)

func TestGetConnections_NoTeamHeader_ReturnsBadRequest(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a regular user and session so auth passes but team header is missing
	user := CreateTestUser(t, db, "noshheader@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/connections/", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d Bad Request when X-Team-Slug header is missing, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if errMsg, ok := body["error"].(string); !ok || !strings.Contains(errMsg, "Team slug") {
		t.Fatalf("expected team slug error in response, got: %v", body)
	}
}

func TestGetConnections_AsTeamUser_ReturnsConnections(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create user and team, add user as team admin
	user := CreateTestUser(t, db, "connuser@example.com", false)
	team, teamUser := CreateTeamAndTeamUser(t, db, "Conn Team", user, "admin")

	// Create a connection directly in DB for this team
	subdomain := "mysubdomain"
	conn := models.NewConnection(models.ConnectionTypeHTTP, &subdomain, teamUser)
	conn.Status = models.ConnectionStatusActive

	if err := db.Create(conn).Error; err != nil {
		t.Fatalf("failed to create connection in DB: %v", err)
	}

	// Create session for user
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/connections/", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for team user getting connections, got %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	bodyStr := string(bodyBytes)

	// Ensure response contains the connection ID we created
	if !strings.Contains(bodyStr, conn.ID) {
		t.Fatalf("expected response to contain connection id %s, response: %s", conn.ID, bodyStr)
	}
}

func TestCreateConnection_HTTP_Success(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a user and team user (the team user provides the secret key)
	user := CreateTestUser(t, db, "creator@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "CreateConn Team", user, "admin")

	// Build request payload using the team user's secret key
	payload := map[string]interface{}{
		"secret_key":      teamUser.SecretKey,
		"connection_type": "http",
		"subdomain":       "uniquesubdomain",
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/connections/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200 OK for successful create, got %d: %s", resp.StatusCode, string(body))
	}

	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	connID, ok := respBody["connection_id"].(string)
	if !ok || connID == "" {
		t.Fatalf("expected connection_id in response, got: %v", respBody)
	}

	// Verify connection exists in DB and has expected fields
	var createdConn models.Connection
	if err := db.Preload("Team").Preload("CreatedBy").Where("id = ?", connID).First(&createdConn).Error; err != nil {
		t.Fatalf("expected connection to be saved in DB: %v", err)
	}

	if createdConn.TeamID != teamUser.Team.ID {
		t.Fatalf("expected connection.TeamID %d to equal teamUser.Team.ID %d", createdConn.TeamID, teamUser.Team.ID)
	}
	if createdConn.CreatedByID != teamUser.ID {
		t.Fatalf("expected connection.CreatedByID %d to equal teamUser.ID %d", createdConn.CreatedByID, teamUser.ID)
	}
	if createdConn.Subdomain == nil || *createdConn.Subdomain != "uniquesubdomain" {
		t.Fatalf("expected subdomain 'uniquesubdomain', got %v", createdConn.Subdomain)
	}
}

func TestCreateConnection_MissingSubdomain_BadRequest(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "nosub@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "NoSub Team", user, "admin")

	payload := map[string]interface{}{
		"secret_key":      teamUser.SecretKey,
		"connection_type": "http",
		// missing subdomain
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/connections/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 400 Bad Request for missing subdomain, got %d: %s", resp.StatusCode, string(body))
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if msg, ok := body["message"].(string); !ok || !strings.Contains(msg, "Subdomain is required") {
		t.Fatalf("expected subdomain required message, got: %v", body)
	}
}

func TestCreateConnection_SecretKeyInvalid_Unauthorized(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	payload := map[string]interface{}{
		"secret_key":      "invalidsecret",
		"connection_type": "http",
		"subdomain":       "some-sub",
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/connections/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 401 Unauthorized for invalid secret key, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestCreateConnection_SubdomainConflict_Conflict(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create an owner user, team and teamUser
	user := CreateTestUser(t, db, "ownerconflict@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "Conflict Team", user, "admin")

	// Create an existing connection using the same subdomain and active status
	sub := "conflictdomain"
	existing := models.NewConnection(models.ConnectionTypeHTTP, &sub, teamUser)
	existing.Status = models.ConnectionStatusActive
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("failed to create existing connection: %v", err)
	}

	// Attempt to create another connection with same subdomain
	payload := map[string]interface{}{
		"secret_key":      teamUser.SecretKey,
		"connection_type": "http",
		"subdomain":       sub,
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/connections/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 409 Conflict for duplicate subdomain, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestCreateConnection_TCP_Success(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a user and team user (the team user provides the secret key)
	user := CreateTestUser(t, db, "tcpcreator@example.com", false)
	_, teamUser := CreateTeamAndTeamUser(t, db, "TCP Team", user, "admin")

	// Build request payload for TCP connection (no subdomain required)
	payload := map[string]interface{}{
		"secret_key":      teamUser.SecretKey,
		"connection_type": "tcp",
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/connections/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200 OK for tcp create, got %d: %s", resp.StatusCode, string(body))
	}

	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	connID, ok := respBody["connection_id"].(string)
	if !ok || connID == "" {
		t.Fatalf("expected connection_id in response for tcp create, got: %v", respBody)
	}

	// Verify DB entry
	var createdConn models.Connection
	if err := db.Where("id = ?", connID).First(&createdConn).Error; err != nil {
		t.Fatalf("expected tcp connection to be saved in DB: %v", err)
	}

	if createdConn.Type != models.ConnectionTypeTCP {
		t.Fatalf("expected connection type %s, got %s", models.ConnectionTypeTCP, createdConn.Type)
	}
}
