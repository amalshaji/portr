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
