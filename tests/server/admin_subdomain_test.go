package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/amalshaji/portr/internal/server/admin/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func TestReservedSubdomainsCreateListAndNormalize(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "reserve-list@example.com", false)
	team, _ := CreateTeamAndTeamUser(t, db, "Reserve List Team", user, models.RoleAdmin)
	session := CreateSessionForUser(t, db, user)

	response := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodPost, "/api/v1/reserved-subdomains/", map[string]string{
		"subdomain": "  My-App  ",
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 201, got %d: %s", response.StatusCode, body)
	}

	var created map[string]any
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created["subdomain"] != "my-app" || created["claim_status"] != "idle" {
		t.Fatalf("unexpected create response: %#v", created)
	}

	list := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodGet, "/api/v1/reserved-subdomains/", nil)
	defer list.Body.Close()
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", list.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(list.Body).Decode(&body); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if body["count"] != float64(1) || body["limit"] != float64(3) || body["base_domain"] != "example.test" {
		t.Fatalf("unexpected list metadata: %#v", body)
	}
}

func TestReservedSubdomainsAreOwnedByExactMembership(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	owner := CreateTestUser(t, db, "reserve-owner@example.com", false)
	team, ownerMembership := CreateTeamAndTeamUser(t, db, "Reservation Owner Team", owner, models.RoleAdmin)
	other := CreateTestUser(t, db, "reserve-other@example.com", false)
	otherMembership := &models.TeamUser{UserID: other.ID, TeamID: team.ID, Role: models.RoleMember}
	if err := db.Create(otherMembership).Error; err != nil {
		t.Fatalf("create second membership: %v", err)
	}

	ownerSession := CreateSessionForUser(t, db, owner)
	otherSession := CreateSessionForUser(t, db, other)
	reserve := reservedSubdomainRequest(t, srv, ownerSession, team.Slug, http.MethodPost, "/api/v1/reserved-subdomains/", map[string]string{"subdomain": "private-name"})
	reserve.Body.Close()
	if reserve.StatusCode != http.StatusCreated {
		t.Fatalf("expected reservation to succeed, got %d", reserve.StatusCode)
	}

	otherList := reservedSubdomainRequest(t, srv, otherSession, team.Slug, http.MethodGet, "/api/v1/reserved-subdomains/", nil)
	defer otherList.Body.Close()
	var listBody map[string]any
	if err := json.NewDecoder(otherList.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode other list: %v", err)
	}
	if listBody["count"] != float64(0) {
		t.Fatalf("expected other membership to have no reservations: %#v", listBody)
	}

	blocked := createHTTPConnectionRequest(t, srv, otherMembership.SecretKey, "private-name")
	defer blocked.Body.Close()
	if blocked.StatusCode != http.StatusConflict {
		t.Fatalf("expected reserved conflict, got %d", blocked.StatusCode)
	}
	var blockedBody map[string]any
	if err := json.NewDecoder(blocked.Body).Decode(&blockedBody); err != nil {
		t.Fatalf("decode blocked response: %v", err)
	}
	if blockedBody["code"] != "reserved_subdomain" || blockedBody["message"] != "This is a reserved subdomain" {
		t.Fatalf("unexpected reserved error: %#v", blockedBody)
	}

	allowed := createHTTPConnectionRequest(t, srv, ownerMembership.SecretKey, "private-name")
	defer allowed.Body.Close()
	if allowed.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(allowed.Body)
		t.Fatalf("expected owner connection to succeed, got %d: %s", allowed.StatusCode, body)
	}
}

func TestReservedSubdomainCanBeReservedAndReleasedWhileActive(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	owner := CreateTestUser(t, db, "active-owner@example.com", false)
	team, ownerMembership := CreateTeamAndTeamUser(t, db, "Active Reservation Team", owner, models.RoleAdmin)
	other := CreateTestUser(t, db, "active-other@example.com", false)
	otherMembership := &models.TeamUser{UserID: other.ID, TeamID: team.ID, Role: models.RoleMember}
	if err := db.Create(otherMembership).Error; err != nil {
		t.Fatalf("create other membership: %v", err)
	}

	subdomain := "already-live"
	connection := models.NewConnection(models.ConnectionTypeHTTP, &subdomain, ownerMembership)
	connection.Status = models.ConnectionStatusActive
	if err := db.Create(connection).Error; err != nil {
		t.Fatalf("create active connection: %v", err)
	}

	session := CreateSessionForUser(t, db, owner)
	reserve := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodPost, "/api/v1/reserved-subdomains/", map[string]string{"subdomain": subdomain})
	defer reserve.Body.Close()
	if reserve.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(reserve.Body)
		t.Fatalf("expected active owner reservation to succeed, got %d: %s", reserve.StatusCode, body)
	}
	var reservation map[string]any
	if err := json.NewDecoder(reserve.Body).Decode(&reservation); err != nil {
		t.Fatalf("decode reservation: %v", err)
	}
	if reservation["claim_status"] != "active" {
		t.Fatalf("expected active claim status, got %#v", reservation)
	}

	release := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodDelete, "/api/v1/reserved-subdomains/"+subdomain, nil)
	release.Body.Close()
	if release.StatusCode != http.StatusNoContent {
		t.Fatalf("expected release to succeed, got %d", release.StatusCode)
	}

	blocked := createHTTPConnectionRequest(t, srv, otherMembership.SecretKey, subdomain)
	blocked.Body.Close()
	if blocked.StatusCode != http.StatusConflict {
		t.Fatalf("expected active tunnel to retain claim, got %d", blocked.StatusCode)
	}

	if err := db.Model(connection).Update("status", models.ConnectionStatusClosed).Error; err != nil {
		t.Fatalf("close connection: %v", err)
	}
	claimed := createHTTPConnectionRequest(t, srv, otherMembership.SecretKey, subdomain)
	defer claimed.Body.Close()
	if claimed.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(claimed.Body)
		t.Fatalf("expected released idle name to be claimable, got %d: %s", claimed.StatusCode, body)
	}
}

func TestReservedSubdomainValidationAndLimit(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	user := CreateTestUser(t, db, "reserve-limit@example.com", false)
	team, _ := CreateTeamAndTeamUser(t, db, "Reservation Limit Team", user, models.RoleAdmin)
	session := CreateSessionForUser(t, db, user)

	invalid := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodPost, "/api/v1/reserved-subdomains/", map[string]string{"subdomain": "not_valid"})
	invalid.Body.Close()
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid subdomain to return 400, got %d", invalid.StatusCode)
	}

	for _, subdomain := range []string{"one", "two", "three"} {
		response := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodPost, "/api/v1/reserved-subdomains/", map[string]string{"subdomain": subdomain})
		response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("expected %s to be reserved, got %d", subdomain, response.StatusCode)
		}
	}

	overLimit := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodPost, "/api/v1/reserved-subdomains/", map[string]string{"subdomain": "four"})
	defer overLimit.Body.Close()
	if overLimit.StatusCode != http.StatusConflict {
		t.Fatalf("expected limit conflict, got %d", overLimit.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(overLimit.Body).Decode(&body); err != nil {
		t.Fatalf("decode limit response: %v", err)
	}
	if body["code"] != "reservation_limit_reached" {
		t.Fatalf("unexpected limit response: %#v", body)
	}
}

func TestReservedSubdomainDatabaseUniquenessIsCaseInsensitive(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	user := CreateTestUser(t, db, "reserve-db@example.com", false)
	_, membership := CreateTeamAndTeamUser(t, db, "Reservation DB Team", user, models.RoleAdmin)
	if err := db.Create(&models.SubdomainReservation{Subdomain: "Case-Test", TeamUserID: membership.ID}).Error; err != nil {
		t.Fatalf("create first reservation: %v", err)
	}
	if err := db.Create(&models.SubdomainReservation{Subdomain: "case-test", TeamUserID: membership.ID}).Error; err == nil {
		t.Fatal("expected case-insensitive duplicate reservation to fail")
	}
}

func TestReservedSubdomainListUsesBoundedQueries(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	user := CreateTestUser(t, db, "reserve-query-count@example.com", false)
	_, membership := CreateTeamAndTeamUser(t, db, "Reservation Query Team", user, models.RoleAdmin)
	for _, subdomain := range []string{"query-one", "query-two", "query-three"} {
		if err := db.Create(&models.SubdomainReservation{Subdomain: subdomain, TeamUserID: membership.ID}).Error; err != nil {
			t.Fatalf("create reservation %s: %v", subdomain, err)
		}
	}
	activeSubdomain := "query-two"
	connection := models.NewConnection(models.ConnectionTypeHTTP, &activeSubdomain, membership)
	connection.Status = models.ConnectionStatusActive
	if err := db.Create(connection).Error; err != nil {
		t.Fatalf("create active connection: %v", err)
	}

	queryCount := 0
	const callbackName = "test:count-reservation-list-queries"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(*gorm.DB) {
		queryCount++
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	defer db.Callback().Query().Remove(callbackName)

	items, err := services.NewSubdomainService(db).List(context.Background(), membership.ID)
	if err != nil {
		t.Fatalf("list reservations: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 reservations, got %d", len(items))
	}
	if queryCount != 2 {
		t.Fatalf("expected reservation list to use 2 queries, got %d", queryCount)
	}
}

func TestReservedSubdomainRaceWithConnectionHasSingleWinner(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	srv := NewTestServer(t, db)

	owner := CreateTestUser(t, db, "race-owner@example.com", false)
	team, _ := CreateTeamAndTeamUser(t, db, "Reservation Race Team", owner, models.RoleAdmin)
	other := CreateTestUser(t, db, "race-other@example.com", false)
	otherMembership := &models.TeamUser{UserID: other.ID, TeamID: team.ID, Role: models.RoleMember}
	if err := db.Create(otherMembership).Error; err != nil {
		t.Fatalf("create other membership: %v", err)
	}
	session := CreateSessionForUser(t, db, owner)

	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		response := reservedSubdomainRequest(t, srv, session, team.Slug, http.MethodPost, "/api/v1/reserved-subdomains/", map[string]string{"subdomain": "race-name"})
		defer response.Body.Close()
		statuses <- response.StatusCode
	}()
	go func() {
		defer wg.Done()
		<-start
		response := createHTTPConnectionRequest(t, srv, otherMembership.SecretKey, "race-name")
		defer response.Body.Close()
		statuses <- response.StatusCode
	}()
	close(start)
	wg.Wait()
	close(statuses)

	successes := 0
	conflicts := 0
	for status := range statuses {
		switch status {
		case http.StatusCreated, http.StatusOK:
			successes++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected concurrent status %d", status)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got %d successes and %d conflicts", successes, conflicts)
	}
}

func reservedSubdomainRequest(t *testing.T, srv interface{ App() *fiber.App }, session *models.Session, teamSlug, method, path string, payload any) *http.Response {
	t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode payload: %v", err)
		}
		body = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Cookie", SessionCookieValue(session))
	req.Header.Set("X-Team-Slug", teamSlug)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := srv.App().Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return response
}

func createHTTPConnectionRequest(t *testing.T, srv interface{ App() *fiber.App }, secretKey, subdomain string) *http.Response {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"secret_key":      secretKey,
		"connection_type": models.ConnectionTypeHTTP,
		"subdomain":       subdomain,
	})
	if err != nil {
		t.Fatalf("encode connection payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	response, err := srv.App().Test(req, -1)
	if err != nil {
		t.Fatalf("connection request failed: %v", err)
	}
	return response
}
