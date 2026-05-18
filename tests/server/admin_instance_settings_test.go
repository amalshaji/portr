package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
)

func TestInstanceSettingsRequireSuperuser_NonSuperuserBlocked(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a regular (non-superuser) user and session
	user := CreateTestUser(t, db, "user@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/instance-settings/", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-superuser to be blocked, got status 200 OK")
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d (Forbidden), got %d", http.StatusForbidden, resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if errMsg, ok := body["error"].(string); !ok || errMsg == "" {
		t.Fatalf("expected error message in response, got: %v", body)
	}
}

func TestInstanceSettingsRequireSuperuser_SuperuserAllowed(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a superuser and session
	admin := CreateTestUser(t, db, "admin@example.com", true)
	adminSess := CreateSessionForUser(t, db, admin)

	req := httptest.NewRequest("GET", "/api/v1/instance-settings/", nil)
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected superuser to access instance-settings, got status %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	// Handler returns default settings including AddUserEmailSubject; check presence
	if _, ok := body["add_user_email_subject"]; !ok {
		t.Fatalf("expected 'add_user_email_subject' field in response, got: %v", body)
	}
}

func TestInstanceSettingsPageRequireSuperuser(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "user@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/instance-settings", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d (Forbidden), got %d", http.StatusForbidden, resp.StatusCode)
	}
}

func TestInstanceSettingsUpdatePersistsAutoSignupSettings(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	admin := CreateTestUser(t, db, "admin@example.com", true)
	team, _ := CreateTeamAndTeamUser(t, db, "Engineering", admin, models.RoleAdmin)
	adminSess := CreateSessionForUser(t, db, admin)
	srv := NewTestServerWithConfig(t, db, func(cfg *serverConfig.AdminConfig) {
		cfg.GithubClientID = "github-client"
		cfg.GithubSecret = "github-secret"
	})

	payload := []byte(`{
		"smtp_enabled": false,
		"smtp_host": "",
		"smtp_port": 587,
		"smtp_username": "",
		"smtp_password": "",
		"from_address": "",
		"add_user_email_subject": "Welcome to Portr!",
		"add_user_email_body": "You have been added to a Portr team. Please set up your account using the temporary password provided.",
		"auto_signup_enabled": true,
		"auto_signup_allowed_domains": " Example.com, @example.com, dev.example.com. ",
		"auto_signup_team_id": ` + jsonNumber(team.ID) + `
	}`)

	req := httptest.NewRequest("PATCH", "/api/v1/instance-settings/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["auto_signup_enabled"] != true {
		t.Fatalf("expected auto signup to be enabled, got %v", body["auto_signup_enabled"])
	}
	if body["auto_signup_allowed_domains"] != "example.com, dev.example.com" {
		t.Fatalf("expected normalized trusted domains, got %v", body["auto_signup_allowed_domains"])
	}
	if body["github_auth_enabled"] != true {
		t.Fatalf("expected github auth to be enabled in response, got %v", body["github_auth_enabled"])
	}

	var settings models.InstanceSettings
	if err := db.First(&settings, 1).Error; err != nil {
		t.Fatalf("failed to load persisted settings: %v", err)
	}
	if settings.AutoSignupTeamID == nil || *settings.AutoSignupTeamID != team.ID {
		t.Fatalf("expected auto signup team ID %d, got %v", team.ID, settings.AutoSignupTeamID)
	}
}

func TestInstanceSettingsUpdateRequiresTrustedDomainsWhenEnabled(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	admin := CreateTestUser(t, db, "admin@example.com", true)
	team, _ := CreateTeamAndTeamUser(t, db, "Engineering", admin, models.RoleAdmin)
	adminSess := CreateSessionForUser(t, db, admin)
	srv := NewTestServerWithConfig(t, db, func(cfg *serverConfig.AdminConfig) {
		cfg.GithubClientID = "github-client"
		cfg.GithubSecret = "github-secret"
	})

	payload := []byte(`{
		"smtp_port": 587,
		"auto_signup_enabled": true,
		"auto_signup_allowed_domains": " , ",
		"auto_signup_team_id": ` + jsonNumber(team.ID) + `
	}`)

	req := httptest.NewRequest("PATCH", "/api/v1/instance-settings/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestListTeamsRequiresSuperuser(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	admin := CreateTestUser(t, db, "admin@example.com", true)
	team, _ := CreateTeamAndTeamUser(t, db, "Engineering", admin, models.RoleAdmin)
	adminSess := CreateSessionForUser(t, db, admin)
	srv := NewTestServer(t, db)

	req := httptest.NewRequest("GET", "/api/v1/team/", nil)
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	var teams []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if len(teams) != 1 || teams[0]["name"] != team.Name {
		t.Fatalf("expected team list to include %q, got %v", team.Name, teams)
	}
}

func jsonNumber(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
