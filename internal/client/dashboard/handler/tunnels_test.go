package handler

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/client/dashboard/service"
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

	svc := service.New(&db.Db{Conn: conn}, nil)
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

func TestRenderResponseAddsSandboxHeadersForHTML(t *testing.T) {
	app, conn := newTestApp(t)
	body := []byte("<html><body>preview</body></html>")
	headers, err := json.Marshal(map[string][]string{
		"Content-Type":   {"text/html; charset=utf-8"},
		"Content-Length": {strconv.Itoa(len(body))},
	})
	if err != nil {
		t.Fatalf("marshal headers: %v", err)
	}

	if err := conn.Create(&db.Request{
		ID:                 "html-response",
		Subdomain:          "alpha",
		Localport:          3000,
		Url:                "/preview",
		Method:             "GET",
		ResponseHeaders:    datatypes.JSON(headers),
		ResponseBody:       body,
		ResponseStatusCode: fiber.StatusOK,
		LoggedAt:           time.Now(),
	}).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := app.Test(httptest.NewRequest("GET", "/api/tunnels/render/html-response?type=response", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Security-Policy"); got != "sandbox" {
		t.Fatalf("expected sandbox CSP header, got %q", got)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}

	rendered, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(rendered) != string(body) {
		t.Fatalf("expected body %q, got %q", string(body), string(rendered))
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
