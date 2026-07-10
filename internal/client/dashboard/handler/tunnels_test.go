package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientconfig "github.com/amalshaji/portr/internal/client/config"
	dashboardservice "github.com/amalshaji/portr/internal/client/dashboard/service"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	conn, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&db.Request{}, &db.WebSocketSession{}, &db.WebSocketEvent{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	svc := dashboardservice.New(&db.Db{Conn: conn}, nil)
	handler := New(nil, svc)

	app := fiber.New()
	handler.RegisterTunnelRoutes(app.Group("/api/tunnels"))
	return app, conn
}

func seedRequests(t *testing.T, conn *gorm.DB, count int) {
	t.Helper()

	base := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	for i := range count {
		if err := conn.Create(&db.Request{
			ID:                 string(rune('a' + i)),
			Subdomain:          "alpha",
			Localport:          3000,
			Url:                "/r",
			Method:             "GET",
			Body:               []byte("request-body"),
			ResponseBody:       []byte("response-body"),
			ResponseStatusCode: 200,
			LoggedAt:           base.Add(time.Duration(i) * time.Minute),
		}).Error; err != nil {
			t.Fatalf("create request: %v", err)
		}
	}
}

func TestGetRequestsPaginationOverHTTP(t *testing.T) {
	app, conn := newTestApp(t)
	seedRequests(t, conn, 5)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/tunnels/alpha/3000?limit=2&offset=2", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var payload struct {
		Requests []map[string]any `json:"requests"`
		Total    int64            `json:"total"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Total != 5 {
		t.Fatalf("expected total 5, got %d", payload.Total)
	}
	if len(payload.Requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(payload.Requests))
	}
	if payload.Requests[0]["ID"] != "c" {
		t.Fatalf("expected request c first on page 2, got %v", payload.Requests[0]["ID"])
	}
	if _, hasBody := payload.Requests[0]["Body"]; hasBody {
		t.Fatalf("list response must not include request bodies")
	}
}

func TestGetRequestByIdOverHTTP(t *testing.T) {
	app, conn := newTestApp(t)
	seedRequests(t, conn, 1)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/tunnels/requests/a", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var payload struct {
		Request map[string]any `json:"request"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Request["ID"] != "a" {
		t.Fatalf("expected request a, got %v", payload.Request["ID"])
	}
	if _, hasBody := payload.Request["Body"]; !hasBody {
		t.Fatalf("detail response must include the request body")
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/api/tunnels/requests/missing", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for missing request, got %d", resp.StatusCode)
	}
}

func TestSerializeWebSocketEventDecodesTextBinaryFrames(t *testing.T) {
	event := db.WebSocketEvent{
		ID:            "evt-1",
		Direction:     "server",
		Opcode:        2,
		OpcodeName:    "binary",
		IsFinal:       true,
		Payload:       []byte(`{"type":"echo","message":"hello"}`),
		PayloadLength: len(`{"type":"echo","message":"hello"}`),
		LoggedAt:      time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC),
	}

	payload := serializeWebSocketEvent(event)
	if payload.PayloadText != `{"type":"echo","message":"hello"}` {
		t.Fatalf("expected payload text to be decoded, got %q", payload.PayloadText)
	}
}

func TestSerializeWebSocketEventLeavesOpaqueBinaryFramesUndecoded(t *testing.T) {
	event := db.WebSocketEvent{
		ID:            "evt-2",
		Direction:     "server",
		Opcode:        2,
		OpcodeName:    "binary",
		IsFinal:       true,
		Payload:       []byte{0xff, 0xfe, 0xfd, 0x00},
		PayloadLength: 4,
		LoggedAt:      time.Date(2026, 3, 29, 12, 5, 0, 0, time.UTC),
	}

	payload := serializeWebSocketEvent(event)
	if payload.PayloadText != "" {
		t.Fatalf("expected no payload text for opaque binary frame, got %q", payload.PayloadText)
	}
}

func TestReplayRequestWithEditsUsesLocalSchemeAndOverrides(t *testing.T) {
	type observedRequest struct {
		Method   string
		Path     string
		Header   string
		ReplayID string
		Body     string
	}

	var observed observedRequest
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read replay body: %v", err)
		}
		observed = observedRequest{
			Method:   r.Method,
			Path:     r.URL.RequestURI(),
			Header:   r.Header.Get("X-Edited"),
			ReplayID: r.Header.Get("X-Portr-Replayed-Request-Id"),
			Body:     string(body),
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer target.Close()

	conn, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&db.Request{}, &db.WebSocketSession{}, &db.WebSocketEvent{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	request := db.Request{
		ID:      "req-edit",
		Host:    strings.TrimPrefix(target.URL, "http://"),
		Url:     "/original",
		Method:  http.MethodPost,
		Headers: datatypes.JSON([]byte(`{"X-Original":["1"]}`)),
		Body:    []byte("original-body"),
	}
	if err := conn.Create(&request).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}

	app := fiber.New()
	cfg := &clientconfig.Config{UseLocalHost: true}
	handler := New(cfg, dashboardservice.New(&db.Db{Conn: conn}, cfg))
	group := app.Group("/api/tunnels")
	handler.RegisterTunnelRoutes(group)

	payload, err := json.Marshal(replayRequestInput{
		Method:       http.MethodPatch,
		Path:         "/edited?x=1",
		Headers:      map[string]string{"X-Edited": "yes"},
		Body:         "edited-body",
		BodyEncoding: "utf8",
	})
	if err != nil {
		t.Fatalf("marshal replay payload: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/tunnels/replay/"+request.ID,
		bytes.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("replay request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(body))
	}

	if observed.Method != http.MethodPatch {
		t.Fatalf("expected edited method PATCH, got %q", observed.Method)
	}
	if observed.Path != "/edited?x=1" {
		t.Fatalf("expected edited path, got %q", observed.Path)
	}
	if observed.Header != "yes" {
		t.Fatalf("expected edited header, got %q", observed.Header)
	}
	if observed.ReplayID != request.ID {
		t.Fatalf("expected replay id %q, got %q", request.ID, observed.ReplayID)
	}
	if observed.Body != "edited-body" {
		t.Fatalf("expected edited body, got %q", observed.Body)
	}
}
