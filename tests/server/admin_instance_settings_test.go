package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
