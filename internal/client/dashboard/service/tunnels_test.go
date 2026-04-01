package service

import (
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/client/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openTestService(t *testing.T) *Service {
	t.Helper()

	conn, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := conn.AutoMigrate(&db.Request{}, &db.WebSocketSession{}, &db.WebSocketEvent{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	return &Service{
		db: &db.Db{Conn: conn},
	}
}

func TestGetTunnelsAggregatesWebSocketActivity(t *testing.T) {
	service := openTestService(t)

	httpLoggedAt := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	wsStartedAt := time.Date(2026, 3, 29, 11, 0, 0, 0, time.UTC)
	wsLastEventAt := time.Date(2026, 3, 29, 11, 5, 0, 0, time.UTC)
	otherLoggedAt := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)

	records := []db.Request{
		{
			ID:                 "req-http",
			Subdomain:          "alpha",
			Localport:          3000,
			Host:               "alpha.example.com",
			Url:                "/http",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           httpLoggedAt,
		},
		{
			ID:                 "req-beta",
			Subdomain:          "beta",
			Localport:          4000,
			Host:               "beta.example.com",
			Url:                "/beta",
			Method:             "POST",
			ResponseStatusCode: 500,
			LoggedAt:           otherLoggedAt,
		},
	}
	if err := service.db.Conn.Create(&records).Error; err != nil {
		t.Fatalf("create requests: %v", err)
	}

	session := db.WebSocketSession{
		ID:               "ws-1",
		Subdomain:        "alpha",
		Localport:        3000,
		Host:             "alpha.example.com",
		URL:              "/socket",
		Method:           "GET",
		StartedAt:        wsStartedAt,
		LastEventAt:      &wsLastEventAt,
		EventCount:       2,
		ClientEventCount: 1,
		ServerEventCount: 1,
	}
	if err := service.db.Conn.Create(&session).Error; err != nil {
		t.Fatalf("create websocket session: %v", err)
	}

	summaries, err := service.GetTunnels()
	if err != nil {
		t.Fatalf("GetTunnels: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("expected 2 tunnel summaries, got %d", len(summaries))
	}

	var alphaSummary *TunnelSummary
	for idx := range summaries {
		if summaries[idx].Subdomain == "alpha" {
			alphaSummary = &summaries[idx]
			break
		}
	}
	if alphaSummary == nil {
		t.Fatalf("alpha tunnel missing from summary")
	}

	if alphaSummary.HTTPRequestCount != 1 {
		t.Fatalf("expected 1 http request, got %d", alphaSummary.HTTPRequestCount)
	}
	if alphaSummary.WebSocketSessionCount != 1 {
		t.Fatalf("expected 1 websocket session, got %d", alphaSummary.WebSocketSessionCount)
	}
	if alphaSummary.ActiveWebSocketCount != 1 {
		t.Fatalf("expected 1 active websocket, got %d", alphaSummary.ActiveWebSocketCount)
	}
	if alphaSummary.LastActivityKind != "websocket" {
		t.Fatalf("expected websocket last activity, got %q", alphaSummary.LastActivityKind)
	}
	if !alphaSummary.LastActivityAt.Equal(wsLastEventAt) {
		t.Fatalf("expected websocket last activity time %v, got %v", wsLastEventAt, alphaSummary.LastActivityAt)
	}
	if alphaSummary.LastMethod != "GET" || alphaSummary.LastStatus != 200 || alphaSummary.LastURL != "/http" {
		t.Fatalf("unexpected latest http summary fields: %#v", alphaSummary)
	}
}

func TestDeleteTunnelLogsRemovesRequestsAndWebSocketData(t *testing.T) {
	service := openTestService(t)

	if err := service.db.Conn.Create(&db.Request{
		ID:        "req-1",
		Subdomain: "alpha",
		Localport: 3000,
		LoggedAt:  time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := service.db.Conn.Create(&db.Request{
		ID:        "req-2",
		Subdomain: "beta",
		Localport: 4000,
		LoggedAt:  time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}

	if err := service.db.Conn.Create(&db.WebSocketSession{
		ID:        "ws-1",
		Subdomain: "alpha",
		Localport: 3000,
		StartedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("create websocket session: %v", err)
	}
	if err := service.db.Conn.Create(&db.WebSocketEvent{
		ID:        "event-1",
		SessionID: "ws-1",
		LoggedAt:  time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("create websocket event: %v", err)
	}

	deletedCount, err := service.DeleteTunnelLogs("alpha", 3000)
	if err != nil {
		t.Fatalf("DeleteTunnelLogs: %v", err)
	}
	if deletedCount != 3 {
		t.Fatalf("expected 3 deleted rows, got %d", deletedCount)
	}

	var remainingAlphaRequests int64
	service.db.Conn.Model(&db.Request{}).Where("subdomain = ? AND localport = ?", "alpha", 3000).Count(&remainingAlphaRequests)
	if remainingAlphaRequests != 0 {
		t.Fatalf("expected alpha requests deleted, got %d remaining", remainingAlphaRequests)
	}

	var remainingAlphaSessions int64
	service.db.Conn.Model(&db.WebSocketSession{}).Where("subdomain = ? AND localport = ?", "alpha", 3000).Count(&remainingAlphaSessions)
	if remainingAlphaSessions != 0 {
		t.Fatalf("expected alpha websocket sessions deleted, got %d remaining", remainingAlphaSessions)
	}

	var remainingAlphaEvents int64
	service.db.Conn.Model(&db.WebSocketEvent{}).Where("session_id = ?", "ws-1").Count(&remainingAlphaEvents)
	if remainingAlphaEvents != 0 {
		t.Fatalf("expected alpha websocket events deleted, got %d remaining", remainingAlphaEvents)
	}

	var remainingBetaRequests int64
	service.db.Conn.Model(&db.Request{}).Where("subdomain = ? AND localport = ?", "beta", 4000).Count(&remainingBetaRequests)
	if remainingBetaRequests != 1 {
		t.Fatalf("expected unrelated beta request to remain, got %d", remainingBetaRequests)
	}
}
