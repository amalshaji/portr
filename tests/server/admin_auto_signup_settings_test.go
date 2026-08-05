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

func TestAutoSignupSettingsRequireSuperuser_NonSuperuserBlocked(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a regular (non-superuser) user and session
	user := CreateTestUser(t, db, "user@example.com", false)
	sess := CreateSessionForUser(t, db, user)

	req := httptest.NewRequest("GET", "/api/v1/auto-signup/", nil)
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

func TestAutoSignupSettingsRequireSuperuser_SuperuserAllowed(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	// Create a superuser and session
	admin := CreateTestUser(t, db, "admin@example.com", true)
	adminSess := CreateSessionForUser(t, db, admin)

	req := httptest.NewRequest("GET", "/api/v1/auto-signup/", nil)
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected superuser to access auto-signup, got status %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if _, ok := body["auto_signup_enabled"]; !ok {
		t.Fatalf("expected 'auto_signup_enabled' field in response, got: %v", body)
	}
	if _, ok := body["auto_signup_domains"]; !ok {
		t.Fatalf("expected 'auto_signup_domains' field in response, got: %v", body)
	}
	if _, ok := body["auto_signup_allowed_domains"]; ok {
		t.Fatalf("did not expect legacy auto signup domain list in response, got: %v", body)
	}
	if _, ok := body["smtp_enabled"]; ok {
		t.Fatalf("did not expect unused SMTP settings in response, got: %v", body)
	}
}

func TestAutoSignupSettingsNoTrailingSlashRoutes(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	admin := CreateTestUser(t, db, "admin-instance-no-slash@example.com", true)
	adminSess := CreateSessionForUser(t, db, admin)

	getReq := httptest.NewRequest("GET", "/api/v1/auto-signup", nil)
	getReq.Header.Set("Cookie", SessionCookieValue(adminSess))

	getResp := DoRequest(t, srv, getReq)
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected GET status 200 OK, got %d", getResp.StatusCode)
	}

	payload := []byte(`{
		"auto_signup_enabled": false,
		"auto_signup_domains": []
	}`)
	patchReq := httptest.NewRequest("PATCH", "/api/v1/auto-signup", bytes.NewReader(payload))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Cookie", SessionCookieValue(adminSess))

	patchResp := DoRequest(t, srv, patchReq)
	defer patchResp.Body.Close()

	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected PATCH status 200 OK, got %d", patchResp.StatusCode)
	}
}

func TestAutoSignupSettingsUpdatePersistsAutoSignupSettings(t *testing.T) {
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
		"auto_signup_enabled": true,
		"auto_signup_domains": [
			{"domain": " Example.com. ", "team_id": ` + jsonNumber(team.ID) + `},
			{"domain": " @dev.example.com. ", "team_id": ` + jsonNumber(team.ID) + `}
		]
	}`)

	req := httptest.NewRequest("PATCH", "/api/v1/auto-signup/", bytes.NewReader(payload))
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
	if body["github_auth_enabled"] != true {
		t.Fatalf("expected github auth to be enabled in response, got %v", body["github_auth_enabled"])
	}
	domains, ok := body["auto_signup_domains"].([]interface{})
	if !ok {
		t.Fatalf("expected auto_signup_domains array, got %T", body["auto_signup_domains"])
	}
	gotDomains := make(map[string]float64, len(domains))
	for _, item := range domains {
		domain, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("expected domain object, got %T", item)
		}
		domainName, ok := domain["domain"].(string)
		if !ok {
			t.Fatalf("expected domain string in %v", domain)
		}
		teamID, ok := domain["team_id"].(float64)
		if !ok {
			t.Fatalf("expected team_id number in %v", domain)
		}
		gotDomains[domainName] = teamID
	}
	if gotDomains["example.com"] != float64(team.ID) || gotDomains["dev.example.com"] != float64(team.ID) {
		t.Fatalf("expected normalized domain mappings for team %d, got %v", team.ID, gotDomains)
	}

	var settings models.AutoSignupSettings
	if err := db.First(&settings, 1).Error; err != nil {
		t.Fatalf("failed to load persisted settings: %v", err)
	}
	if !settings.AutoSignupEnabled {
		t.Fatalf("expected persisted auto signup enabled")
	}

	var mappings []models.AutoSignupDomain
	if err := db.Order("domain ASC").Find(&mappings).Error; err != nil {
		t.Fatalf("failed to load persisted auto signup domains: %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("expected 2 persisted auto signup domains, got %d", len(mappings))
	}
	for _, mapping := range mappings {
		if mapping.TeamID != team.ID {
			t.Fatalf("expected domain %q to map to team %d, got %d", mapping.Domain, team.ID, mapping.TeamID)
		}
	}
}

func TestAutoSignupSettingsUpdateRequiresTrustedDomainsWhenEnabled(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	admin := CreateTestUser(t, db, "admin@example.com", true)
	adminSess := CreateSessionForUser(t, db, admin)
	srv := NewTestServerWithConfig(t, db, func(cfg *serverConfig.AdminConfig) {
		cfg.GithubClientID = "github-client"
		cfg.GithubSecret = "github-secret"
	})

	payload := []byte(`{
		"auto_signup_enabled": true,
		"auto_signup_domains": []
	}`)

	req := httptest.NewRequest("PATCH", "/api/v1/auto-signup/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestAutoSignupSettingsUpdateReassignsExistingDomain(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	admin := CreateTestUser(t, db, "admin@example.com", true)
	amalTeam, _ := CreateTeamAndTeamUser(t, db, "Amal", admin, models.RoleAdmin)
	otherTeam, _ := CreateTeamAndTeamUser(t, db, "Other", admin, models.RoleAdmin)
	adminSess := CreateSessionForUser(t, db, admin)
	srv := NewTestServerWithConfig(t, db, func(cfg *serverConfig.AdminConfig) {
		cfg.GithubClientID = "github-client"
		cfg.GithubSecret = "github-secret"
	})

	settings := models.DefaultAutoSignupSettings()
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("failed to create auto signup settings: %v", err)
	}
	if err := db.Create(&models.AutoSignupDomain{Domain: "amal.sh", TeamID: amalTeam.ID}).Error; err != nil {
		t.Fatalf("failed to create existing auto signup domain: %v", err)
	}

	payload := []byte(`{
		"auto_signup_enabled": true,
		"auto_signup_domains": [
			{"domain": "amal.sh", "team_id": ` + jsonNumber(otherTeam.ID) + `}
		]
	}`)

	req := httptest.NewRequest("PATCH", "/api/v1/auto-signup/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	var mapping models.AutoSignupDomain
	if err := db.Where("domain = ?", "amal.sh").First(&mapping).Error; err != nil {
		t.Fatalf("failed to load reassigned domain: %v", err)
	}
	if mapping.TeamID != otherTeam.ID {
		t.Fatalf("expected domain to move to team %d, got %d", otherTeam.ID, mapping.TeamID)
	}
}

func TestAutoSignupSettingsUpdateRejectsDuplicateDomainMappings(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	admin := CreateTestUser(t, db, "admin@example.com", true)
	amalTeam, _ := CreateTeamAndTeamUser(t, db, "Amal", admin, models.RoleAdmin)
	otherTeam, _ := CreateTeamAndTeamUser(t, db, "Other", admin, models.RoleAdmin)
	adminSess := CreateSessionForUser(t, db, admin)
	srv := NewTestServerWithConfig(t, db, func(cfg *serverConfig.AdminConfig) {
		cfg.GithubClientID = "github-client"
		cfg.GithubSecret = "github-secret"
	})

	payload := []byte(`{
		"auto_signup_enabled": true,
		"auto_signup_domains": [
			{"domain": "amal.sh", "team_id": ` + jsonNumber(amalTeam.ID) + `},
			{"domain": "@amal.sh.", "team_id": ` + jsonNumber(otherTeam.ID) + `}
		]
	}`)

	req := httptest.NewRequest("PATCH", "/api/v1/auto-signup/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SessionCookieValue(adminSess))

	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != "Domain amal.sh is already configured for another team" {
		t.Fatalf("expected duplicate domain team error, got %v", body)
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
