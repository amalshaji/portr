package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequireAuth_MissingSession_Unauthorized(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	req := httptest.NewRequest("GET", "/api/v1/user/me/teams", nil)
	// no cookie
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_InvalidSession_Unauthorized(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	req := httptest.NewRequest("GET", "/api/v1/user/me/teams", nil)
	req.Header.Set("Cookie", "portr_session=invalidtoken")
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid token, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_ExpiredSession_Unauthorized(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "expired@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	// Set session expiry in the past
	if err := db.Model(sess).Update("expires_at", time.Now().Add(-1*time.Hour)).Error; err != nil {
		t.Fatalf("failed to expire session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/user/me/teams", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired session, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_ValidSession_AllowsRequest(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "valid@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/user/me/teams", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for valid session, got %d", resp.StatusCode)
	}

	// Expect an array (possibly empty) in response
	var teams []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

func TestRequireTeamUser_MissingTeamHeader_BadRequest(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "noteamheader@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	// no X-Team-Slug
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when team header missing, got %d", resp.StatusCode)
	}
}

func TestRequireTeamUser_UserNotInTeam_Forbidden(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "notinteam@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	// create a team but do NOT add the user to it
	team, _ := CreateTeamAndTeamUser(t, db, "Other Team", CreateTestUser(t, db, "other@example.com", false), "admin")

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", team.Slug)
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when user not in team, got %d", resp.StatusCode)
	}
}

func TestRequireTeamUser_UserInTeam_AllowsRequest(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "member@example.com", false)
	_, _ = CreateTeamAndTeamUser(t, db, "My Team", user, "admin")
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", "my-team") // slug generated from "My Team"
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for team user, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["team"] == nil {
		t.Fatalf("expected team data in response, got %+v", body)
	}
}

func TestRequireAdmin_MemberForbidden_AdminAllowed(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// member user
	member := CreateTestUser(t, db, "simplemember@example.com", false)
	_, memberTeamUser := CreateTeamAndTeamUser(t, db, "AdminTestTeam", member, "member")
	memberSess := CreateSessionForUser(t, db, member)

	// Try admin-only endpoint as member: POST /api/v1/team/add
	payload := `{"email":"newuser@example.com","role":"member"}`
	reqMember := httptest.NewRequest("POST", "/api/v1/team/add", strings.NewReader(payload))
	reqMember.Header.Set("Cookie", SessionCookieValue(memberSess))
	reqMember.Header.Set("X-Team-Slug", memberTeamUser.Team.Slug)
	reqMember.Header.Set("Content-Type", "application/json")
	respMember := DoRequest(t, srv, reqMember)
	defer respMember.Body.Close()

	if respMember.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for member calling admin endpoint, got %d", respMember.StatusCode)
	}

	// admin user
	admin := CreateTestUser(t, db, "adminuser@example.com", false)
	_, adminTeamUser := CreateTeamAndTeamUser(t, db, "AdminTestTeam2", admin, "admin")
	adminSess := CreateSessionForUser(t, db, admin)

	reqAdmin := httptest.NewRequest("POST", "/api/v1/team/add", strings.NewReader(payload))
	reqAdmin.Header.Set("Cookie", SessionCookieValue(adminSess))
	reqAdmin.Header.Set("X-Team-Slug", adminTeamUser.Team.Slug)
	reqAdmin.Header.Set("Content-Type", "application/json")
	respAdmin := DoRequest(t, srv, reqAdmin)
	defer respAdmin.Body.Close()

	if respAdmin.StatusCode != http.StatusOK {
		// The handler may return 200 on success; if input validation fails it could return 400.
		t.Fatalf("expected 200 OK for admin calling admin endpoint, got %d", respAdmin.StatusCode)
	}
}

func TestRequireSuperuser_NonSuperuserForbidden_SuperuserAllowed(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// non-superuser
	user := CreateTestUser(t, db, "normal@example.com", false)
	userSess := CreateSessionForUser(t, db, user)

	payload := `{"name":"New Team From Test"}`
	req := httptest.NewRequest("POST", "/api/v1/team/", strings.NewReader(payload))
	req.Header.Set("Cookie", SessionCookieValue(userSess))
	req.Header.Set("Content-Type", "application/json")
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-superuser creating team, got %d", resp.StatusCode)
	}

	// superuser
	super := CreateTestUser(t, db, "super@example.com", true)
	superSess := CreateSessionForUser(t, db, super)

	reqSuper := httptest.NewRequest("POST", "/api/v1/team/", strings.NewReader(payload))
	reqSuper.Header.Set("Cookie", SessionCookieValue(superSess))
	reqSuper.Header.Set("Content-Type", "application/json")
	respSuper := DoRequest(t, srv, reqSuper)
	defer respSuper.Body.Close()

	if respSuper.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for superuser creating team, got %d", respSuper.StatusCode)
	}
}
