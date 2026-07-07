package ssh

import (
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

func newLoggingTestClient(store *clientdb.Db, enabled bool) *SshClient {
	return &SshClient{
		config: clientcfg.ClientConfig{
			EnableRequestLogging: enabled,
			Tunnel: clientcfg.Tunnel{
				Name:      "test-server",
				Subdomain: "test-server",
				Port:      8010,
			},
		},
		db: store,
	}
}

func newHTTPLogFixtures() (*http.Request, *http.Response) {
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

	return request, response
}

func newWebSocketLogFixtures() (*http.Request, *http.Response) {
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

	return request, response
}

func TestLogHttpRequestDoesNotPersistWhenRequestLoggingDisabled(t *testing.T) {
	store := newTestRequestStore(t)
	client := newLoggingTestClient(store, false)
	request, response := newHTTPLogFixtures()

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

func TestLogHttpRequestPersistsWhenRequestLoggingEnabled(t *testing.T) {
	store := newTestRequestStore(t)
	client := newLoggingTestClient(store, true)
	request, response := newHTTPLogFixtures()

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
	if count != 1 {
		t.Fatalf("expected 1 stored request, got %d", count)
	}
}

func TestLogWebSocketSessionDoesNotPersistWhenRequestLoggingDisabled(t *testing.T) {
	store := newTestRequestStore(t)
	client := newLoggingTestClient(store, false)
	request, response := newWebSocketLogFixtures()

	sessionID := client.logWebSocketSession("handshake-1", request, response)
	if sessionID != "" {
		t.Fatalf("expected no websocket session id when logging is disabled, got %q", sessionID)
	}

	client.recordWebSocketEvent("session-1", "client", &webSocketFrame{
		Opcode:        1,
		IsFinal:       true,
		Payload:       []byte("hello"),
		PayloadLength: len("hello"),
	})

	var sessionCount int64
	if err := store.Conn.Model(&clientdb.WebSocketSession{}).Count(&sessionCount).Error; err != nil {
		t.Fatalf("count websocket sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("expected no stored websocket sessions, got %d", sessionCount)
	}

	var eventCount int64
	if err := store.Conn.Model(&clientdb.WebSocketEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count websocket events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected no stored websocket events, got %d", eventCount)
	}
}

func TestLogWebSocketSessionPersistsWhenRequestLoggingEnabled(t *testing.T) {
	store := newTestRequestStore(t)
	client := newLoggingTestClient(store, true)
	request, response := newWebSocketLogFixtures()

	sessionID := client.logWebSocketSession("handshake-1", request, response)
	if sessionID == "" {
		t.Fatal("expected websocket session id to be created")
	}

	client.recordWebSocketEvent(sessionID, "client", &webSocketFrame{
		Opcode:        1,
		IsFinal:       true,
		Payload:       []byte("hello"),
		PayloadLength: len("hello"),
	})

	var sessionCount int64
	if err := store.Conn.Model(&clientdb.WebSocketSession{}).Count(&sessionCount).Error; err != nil {
		t.Fatalf("count websocket sessions: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("expected 1 stored websocket session, got %d", sessionCount)
	}

	var eventCount int64
	if err := store.Conn.Model(&clientdb.WebSocketEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count websocket events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("expected 1 stored websocket event, got %d", eventCount)
	}
}
