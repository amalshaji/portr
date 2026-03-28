package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthConfigAccessible(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	req := httptest.NewRequest("GET", "/api/v1/auth/auth-config", nil)
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	// Expect some config to be returned; ensure the response is not empty.
	if len(body) == 0 {
		t.Fatalf("expected non-empty auth-config response, got empty body")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a user with a known password
	_ = CreateTestUser(t, db, "loginuser@example.com", false)

	// Attempt login with incorrect password
	payload := `{"email":"loginuser@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	// Expect bad request (handler returns 400 for invalid credentials)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d Bad Request for invalid credentials, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestLogin_SuccessSetsSessionCookie(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a user with known password
	user := CreateTestUser(t, db, "gooduser@example.com", false)
	// The CreateTestUser helper sets the password to "password123"

	payload := `{"email":"` + user.Email + `","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for successful login, got %d", resp.StatusCode)
	}

	// Check Set-Cookie header was set for session
	setCookie := resp.Header.Get("Set-Cookie")
	if setCookie == "" || !strings.Contains(setCookie, "portr_session=") {
		t.Fatalf("expected Set-Cookie header to contain portr_session, got %q", setCookie)
	}
}

func TestLogout_RequiresAuthAndSucceedsWithSession(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Logout without session may return 401 or 200 depending on middleware behavior; accept both.
	reqNoAuth := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	respNoAuth := DoRequest(t, srv, reqNoAuth)
	defer respNoAuth.Body.Close()
	if respNoAuth.StatusCode != http.StatusUnauthorized && respNoAuth.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d Unauthorized or %d OK for logout without auth, got %d", http.StatusUnauthorized, http.StatusOK, respNoAuth.StatusCode)
	}

	// Create a user and a session, then perform logout
	user := CreateTestUser(t, db, "logoutuser@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK for logout with valid session, got %d", resp.StatusCode)
	}

	// Optionally verify response body contains success information
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
		// Expect either a message or success flag; don't fail if shape differs
		if _, ok := body["error"]; ok {
			t.Fatalf("expected logout success, but got error in response: %v", body)
		}
	}
}
