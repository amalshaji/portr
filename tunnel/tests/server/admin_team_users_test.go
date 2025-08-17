package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
)

func TestListTeamUsers_ReturnsUsers(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	owner := CreateTestUser(t, db, "listowner@example.com", false)
	_, ownerTeamUser := CreateTeamAndTeamUser(t, db, "List Team", owner, "admin")

	member := CreateTestUser(t, db, "listmember@example.com", false)
	_, _ = CreateTeamAndTeamUser(t, db, "List Team Member", member, "member")
	// Add the member to the same team as owner to ensure same team membership
	teamUserForMember := &models.TeamUser{
		UserID: member.ID,
		TeamID: ownerTeamUser.TeamID,
		Role:   models.RoleMember,
	}
	if err := db.Create(teamUserForMember).Error; err != nil {
		t.Fatalf("failed to create team user for member: %v", err)
	}

	sess := CreateSessionForUser(t, db, owner)

	req := httptest.NewRequest("GET", "/api/v1/team/users", nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", ownerTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for list team users, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	countF, ok := body["count"].(float64)
	if !ok {
		t.Fatalf("expected numeric count in response, got: %v", body["count"])
	}
	if int(countF) < 2 {
		t.Fatalf("expected at least 2 team users, got %v", countF)
	}

	data, ok := body["data"].([]interface{})
	if !ok || len(data) == 0 {
		t.Fatalf("expected non-empty data array, got: %v", body["data"])
	}

	// Ensure emails present
	foundOwner, foundMember := false, false
	for _, item := range data {
		if m, ok := item.(map[string]interface{}); ok {
			if userObj, ok := m["user"].(map[string]interface{}); ok {
				if userObj["email"] == owner.Email {
					foundOwner = true
				}
				if userObj["email"] == member.Email {
					foundMember = true
				}
			}
		}
	}
	if !foundOwner || !foundMember {
		t.Fatalf("expected both owner and member emails in response, got: owner=%v member=%v", foundOwner, foundMember)
	}
}

func TestRemoveUser_AsAdmin_RemovesTeamUserAndUserDeletedIfNoOtherTeams(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	owner := CreateTestUser(t, db, "removeowner@example.com", false)
	_, ownerTeamUser := CreateTeamAndTeamUser(t, db, "Remove Team", owner, "admin")

	member := CreateTestUser(t, db, "removemember@example.com", false)
	// Add the member directly to the owner's team (avoid creating a separate team)
	teamUserForMember := &models.TeamUser{
		UserID: member.ID,
		TeamID: ownerTeamUser.TeamID,
		Role:   models.RoleMember,
	}
	if err := db.Create(teamUserForMember).Error; err != nil {
		t.Fatalf("failed to create team user for member: %v", err)
	}

	// Create session for owner (admin)
	sess := CreateSessionForUser(t, db, owner)

	// Remove the member's TeamUser record using its ID
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/team/users/%d", teamUserForMember.ID), nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", ownerTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK when removing user as admin, got %d", resp.StatusCode)
	}

	// Verify team_user deleted
	var tu models.TeamUser
	if err := db.Where("id = ?", teamUserForMember.ID).First(&tu).Error; err == nil {
		t.Fatalf("expected team_user to be deleted, but found record: %v", tu)
	}

	// Since member has no other teams, user should be deleted
	var u models.User
	if err := db.Where("id = ?", member.ID).First(&u).Error; err == nil {
		t.Fatalf("expected user to be deleted after removing from last team, but found user: %v", u)
	}
}

func TestRemoveUser_CannotRemoveSuperuserByNonSuperuser(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// owner is admin but not superuser
	owner := CreateTestUser(t, db, "nonrootadmin@example.com", false)
	_, ownerTeamUser := CreateTeamAndTeamUser(t, db, "NoRemove Team", owner, "admin")

	// target is superuser
	super := CreateTestUser(t, db, "targetsuper@example.com", true)
	// create team_user linking super to the same team
	superTeamUser := &models.TeamUser{
		UserID: super.ID,
		TeamID: ownerTeamUser.TeamID,
		Role:   models.RoleMember,
	}
	if err := db.Create(superTeamUser).Error; err != nil {
		t.Fatalf("failed to create team_user for superuser: %v", err)
	}

	sess := CreateSessionForUser(t, db, owner)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/team/users/%d", superTeamUser.ID), nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", ownerTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when non-superuser tries to remove a superuser, got %d", resp.StatusCode)
	}
}

func TestResetPassword_BySuperuser_SucceedsAndPersists(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	super := CreateTestUser(t, db, "resetsuper@example.com", true)
	_, superTeamUser := CreateTeamAndTeamUser(t, db, "Reset Team", super, "admin")

	member := CreateTestUser(t, db, "resetmember@example.com", false)
	// add member to same team
	memberTeamUser := &models.TeamUser{
		UserID: member.ID,
		TeamID: superTeamUser.TeamID,
		Role:   models.RoleMember,
	}
	if err := db.Create(memberTeamUser).Error; err != nil {
		t.Fatalf("failed to create team_user for member: %v", err)
	}

	sess := CreateSessionForUser(t, db, super)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/team/users/%d/reset-password", memberTeamUser.ID), nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", superTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK when superuser resets password, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	pw, ok := body["password"].(string)
	if !ok || pw == "" {
		t.Fatalf("expected password string in response, got: %v", body)
	}

	// Verify persisted password by reloading user and checking the password
	var persisted models.User
	if err := db.First(&persisted, member.ID).Error; err != nil {
		t.Fatalf("failed to reload user from DB: %v", err)
	}
	if !persisted.CheckPassword(pw) {
		t.Fatalf("expected persisted user's password to match returned password")
	}
}

func TestResetPassword_ByNonSuperuser_Forbidden(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// admin is not a superuser
	admin := CreateTestUser(t, db, "adminnotsuper@example.com", false)
	_, adminTeamUser := CreateTeamAndTeamUser(t, db, "ResetForbidden Team", admin, "admin")

	target := CreateTestUser(t, db, "targetnoforce@example.com", false)
	// link target to same team
	targetTeamUser := &models.TeamUser{
		UserID: target.ID,
		TeamID: adminTeamUser.TeamID,
		Role:   models.RoleMember,
	}
	if err := db.Create(targetTeamUser).Error; err != nil {
		t.Fatalf("failed to create team_user for target: %v", err)
	}

	sess := CreateSessionForUser(t, db, admin)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/team/users/%d/reset-password", targetTeamUser.ID), nil)
	req.Header.Set("Cookie", SessionCookieValue(sess))
	req.Header.Set("X-Team-Slug", adminTeamUser.Team.Slug)

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when non-superuser tries to reset password, got %d", resp.StatusCode)
	}
}
