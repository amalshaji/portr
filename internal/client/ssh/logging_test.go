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

func TestLogHttpRequestPersistsWhenRequestLoggingDisabled(t *testing.T) {
	store := newTestRequestStore(t)
	client := &SshClient{
		config: clientcfg.ClientConfig{
			EnableRequestLogging: false,
			Tunnel: clientcfg.Tunnel{
				Name:      "test-server",
				Subdomain: "test-server",
				Port:      8010,
			},
		},
		db: store,
	}

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
	)

	var count int64
	if err := store.Conn.Model(&clientdb.Request{}).Count(&count).Error; err != nil {
		t.Fatalf("count requests: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 stored request, got %d", count)
	}
}

func TestLogWebSocketSessionPersistsWhenRequestLoggingDisabled(t *testing.T) {
	store := newTestRequestStore(t)
	client := &SshClient{
		config: clientcfg.ClientConfig{
			EnableRequestLogging: false,
			Tunnel: clientcfg.Tunnel{
				Name:      "test-server",
				Subdomain: "test-server",
				Port:      8010,
			},
		},
		db: store,
	}

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
	if sessionID == "" {
		t.Fatal("expected websocket session id to be created")
	}

	var count int64
	if err := store.Conn.Model(&clientdb.WebSocketSession{}).Count(&count).Error; err != nil {
		t.Fatalf("count websocket sessions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 stored websocket session, got %d", count)
	}
}
