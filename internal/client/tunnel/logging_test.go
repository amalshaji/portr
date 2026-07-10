package tunnel

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	clientdb "github.com/amalshaji/portr/internal/client/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestRequestStore(t *testing.T) *clientdb.Db {
	t.Helper()

	path := filepath.Join(t.TempDir(), "db.sqlite")
	conn, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := conn.AutoMigrate(
		&clientdb.Request{},
		&clientdb.WebSocketSession{},
		&clientdb.WebSocketEvent{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return &clientdb.Db{Conn: conn}
}

func newLoggingTestClient(t *testing.T, enabled bool) (*Client, *clientdb.Db) {
	t.Helper()

	store := newTestRequestStore(t)
	client := &Client{
		config: clientcfg.ClientConfig{
			EnableRequestLogging: enabled,
			RedactHeaders:        append([]string(nil), clientcfg.DefaultRedactHeaders...),
			Tunnel: clientcfg.Tunnel{
				Name:      "test-server",
				Subdomain: "test-server",
				Port:      8010,
			},
		},
		db: store,
	}

	return client, store
}

func TestLogHttpRequestSkipsPersistenceWhenRequestLoggingDisabled(t *testing.T) {
	client, store := newLoggingTestClient(t, false)

	request := httptest.NewRequest(
		http.MethodPost,
		"https://test-server.go-v1.portr.dev/requests/json",
		strings.NewReader(`{"ok":true}`),
	)
	request.Host = "test-server.go-v1.portr.dev"
	request.Header.Set("Content-Type", "application/json")

	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	client.logHttpRequest(
		"req-1",
		request,
		[]byte(`{"ok":true}`),
		response,
		[]byte(`{"saved":true}`),
		42,
	)

	var count int64
	if err := store.Conn.Model(&clientdb.Request{}).Count(&count).Error; err != nil {
		t.Fatalf("count requests: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no stored requests, got %d", count)
	}
}

func TestLogWebSocketSessionSkipsPersistenceWhenRequestLoggingDisabled(t *testing.T) {
	client, store := newLoggingTestClient(t, false)

	request := httptest.NewRequest(
		http.MethodGet,
		"https://test-server.go-v1.portr.dev/ws/echo",
		nil,
	)
	request.Host = "test-server.go-v1.portr.dev"
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	request.Header.Set("Sec-WebSocket-Version", "13")

	response := &http.Response{
		StatusCode: http.StatusSwitchingProtocols,
		Header: http.Header{
			"Connection": []string{"Upgrade"},
			"Upgrade":    []string{"websocket"},
		},
	}

	sessionID := client.logWebSocketSession("handshake-1", request, response)
	if sessionID != "" {
		t.Fatalf("expected no websocket session id when logging disabled, got %q", sessionID)
	}

	var count int64
	if err := store.Conn.Model(&clientdb.WebSocketSession{}).Count(&count).Error; err != nil {
		t.Fatalf("count websocket sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no stored websocket sessions, got %d", count)
	}
}

func TestLogHttpRequestRedactsSensitiveHeaders(t *testing.T) {
	client, store := newLoggingTestClient(t, true)

	request := httptest.NewRequest(http.MethodGet, "https://test-server.go-v1.portr.dev/secret", nil)
	request.Host = "test-server.go-v1.portr.dev"
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Cookie", "session=secret")
	request.Header.Set("X-Trace", "visible")

	response := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Set-Cookie": []string{"session=secret"},
			"X-Trace":    []string{"visible"},
		},
	}

	client.logHttpRequest("req-1", request, nil, response, nil, 0)

	var stored clientdb.Request
	if err := store.Conn.First(&stored, "id = ?", "req-1").Error; err != nil {
		t.Fatalf("load stored request: %v", err)
	}

	var requestHeaders map[string][]string
	if err := json.Unmarshal(stored.Headers, &requestHeaders); err != nil {
		t.Fatalf("unmarshal request headers: %v", err)
	}
	if got := requestHeaders["Authorization"]; len(got) != 1 || got[0] != "[redacted]" {
		t.Fatalf("expected Authorization to be redacted, got %v", got)
	}
	if got := requestHeaders["Cookie"]; len(got) != 1 || got[0] != "[redacted]" {
		t.Fatalf("expected Cookie to be redacted, got %v", got)
	}
	if got := requestHeaders["X-Trace"]; len(got) != 1 || got[0] != "visible" {
		t.Fatalf("expected X-Trace to remain visible, got %v", got)
	}

	var responseHeaders map[string][]string
	if err := json.Unmarshal(stored.ResponseHeaders, &responseHeaders); err != nil {
		t.Fatalf("unmarshal response headers: %v", err)
	}
	if got := responseHeaders["Set-Cookie"]; len(got) != 1 || got[0] != "[redacted]" {
		t.Fatalf("expected Set-Cookie to be redacted, got %v", got)
	}
	if got := responseHeaders["X-Trace"]; len(got) != 1 || got[0] != "visible" {
		t.Fatalf("expected response X-Trace to remain visible, got %v", got)
	}
}

func TestLogWebSocketSessionRedactsSensitiveHeaders(t *testing.T) {
	client, store := newLoggingTestClient(t, true)

	request := httptest.NewRequest(http.MethodGet, "https://test-server.go-v1.portr.dev/ws/echo", nil)
	request.Host = "test-server.go-v1.portr.dev"
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Cookie", "session=secret")
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")

	response := &http.Response{
		StatusCode: http.StatusSwitchingProtocols,
		Header: http.Header{
			"Set-Cookie": []string{"session=secret"},
			"Upgrade":    []string{"websocket"},
		},
	}

	sessionID := client.logWebSocketSession("handshake-1", request, response)
	if sessionID == "" {
		t.Fatal("expected websocket session id")
	}

	var stored clientdb.WebSocketSession
	if err := store.Conn.First(&stored, "id = ?", sessionID).Error; err != nil {
		t.Fatalf("load stored websocket session: %v", err)
	}

	var requestHeaders map[string][]string
	if err := json.Unmarshal(stored.RequestHeaders, &requestHeaders); err != nil {
		t.Fatalf("unmarshal request headers: %v", err)
	}
	if got := requestHeaders["Authorization"]; len(got) != 1 || got[0] != "[redacted]" {
		t.Fatalf("expected Authorization to be redacted, got %v", got)
	}
	if got := requestHeaders["Cookie"]; len(got) != 1 || got[0] != "[redacted]" {
		t.Fatalf("expected Cookie to be redacted, got %v", got)
	}

	var responseHeaders map[string][]string
	if err := json.Unmarshal(stored.ResponseHeaders, &responseHeaders); err != nil {
		t.Fatalf("unmarshal response headers: %v", err)
	}
	if got := responseHeaders["Set-Cookie"]; len(got) != 1 || got[0] != "[redacted]" {
		t.Fatalf("expected Set-Cookie to be redacted, got %v", got)
	}
	if got := responseHeaders["Upgrade"]; len(got) != 1 || got[0] != "websocket" {
		t.Fatalf("expected Upgrade to remain visible, got %v", got)
	}
}

func TestRetryTransientSQLiteWriteRetriesLockedErrors(t *testing.T) {
	attempts := 0
	err := retryTransientSQLiteWrite(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("database table is locked")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected retry to recover from transient sqlite lock: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
