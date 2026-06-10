package appserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	tunnelclient "github.com/amalshaji/portr/internal/client/tunnel"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/charmbracelet/log"
)

func TestNormalizeCallbackURLsDeduplicates(t *testing.T) {
	got := normalizeCallbackURLs("http://example.test/a", []string{"http://example.test/a", " http://example.test/b "})
	if len(got) != 2 {
		t.Fatalf("expected 2 callback URLs, got %#v", got)
	}
	if got[0] != "http://example.test/a" || got[1] != "http://example.test/b" {
		t.Fatalf("unexpected callback URLs: %#v", got)
	}
}

func TestValidateTunnelRequestRejectsInvalidPort(t *testing.T) {
	tunnel := clientcfg.Tunnel{
		Type: constants.Http,
		Port: 0,
	}
	tunnel.SetDefaults()

	if err := validateTunnelRequest(tunnel, "", nil); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestValidateTunnelRequestRejectsInvalidCallbackURL(t *testing.T) {
	tunnel := clientcfg.Tunnel{
		Type: constants.Tcp,
		Port: 5432,
	}
	tunnel.SetDefaults()

	if err := validateTunnelRequest(tunnel, "ftp://example.test/hook", nil); err == nil {
		t.Fatal("expected invalid callback URL error")
	}
}

func TestSupportsHTTPPoolingVersion(t *testing.T) {
	if !supportsHTTPPoolingVersion("v1.0.0") {
		t.Fatal("expected v1.0.0 to support HTTP pooling")
	}
	if supportsHTTPPoolingVersion("0.9.9") {
		t.Fatal("expected 0.9.9 to disable HTTP pooling")
	}
	if supportsHTTPPoolingVersion("not-a-version") {
		t.Fatal("expected invalid versions to disable HTTP pooling")
	}
}

func TestClientConfigForTunnelDisablesTunnelClientTerminalLogs(t *testing.T) {
	cfg := clientcfg.Config{}
	cfg.SetDefaults()

	manager := NewManager(cfg, nil)
	tunnel := clientcfg.Tunnel{
		Type: constants.Http,
		Host: "localhost",
		Port: 3000,
	}
	tunnel.SetDefaults()

	clientConfig := manager.clientConfigForTunnel(tunnel)
	if !clientConfig.DisableTerminalLogs {
		t.Fatal("expected app-server tunnel client terminal logs to be disabled")
	}
}

func TestPrepareStubTunnelUsesSharedLocalResponder(t *testing.T) {
	cfg := clientcfg.Config{}
	cfg.SetDefaults()

	manager := NewManager(cfg, nil)
	t.Cleanup(func() {
		manager.Shutdown(context.Background())
	})

	first := clientcfg.Tunnel{
		Type:             constants.Stub,
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"name":"{{name}}"}`,
	}
	second := clientcfg.Tunnel{
		Type:             constants.Stub,
		Subdomain:        "yaml",
		ResponseFormat:   "application/yml",
		ResponseTemplate: "name: {{name}}\n",
	}

	if err := manager.prepareStubTunnel(&first); err != nil {
		t.Fatalf("prepare first stub tunnel: %v", err)
	}
	if err := manager.prepareStubTunnel(&second); err != nil {
		t.Fatalf("prepare second stub tunnel: %v", err)
	}

	if first.Port == 0 || second.Port == 0 || first.Port != second.Port {
		t.Fatalf("expected stub tunnels to share one responder port, got %d and %d", first.Port, second.Port)
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/?name=portr", first.Port), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "yaml.example.test"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request stub responder: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != "name: portr\n" {
		t.Fatalf("unexpected response body: %q", string(body))
	}
}

func TestRecordEventLogsTerminalEvent(t *testing.T) {
	var output bytes.Buffer
	manager := &Manager{
		logger: log.NewWithOptions(&output, log.Options{}),
		events: make([]TunnelEvent, 0),
	}
	tunnel := &tunnelRuntime{
		id: "01HZYR9VK2C0G6KTX4TK9Z5T6S",
		status: TunnelStatus{
			Name:      "web",
			Type:      constants.Http,
			Host:      "localhost",
			Port:      3000,
			Subdomain: "my-app",
			TunnelURL: "https://my-app.example.com",
		},
	}

	manager.recordEvent(tunnel, tunnelclient.Event{
		Type: tunnelclient.EventStarted,
		At:   time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC),
	})

	logLine := output.String()
	for _, want := range []string{
		"App-server tunnel event",
		"event=started",
		"tunnel_id=01HZYR9VK2C0G6KTX4TK9Z5T6S",
		"connection_type=http",
		"host=localhost",
		"port=3000",
		"name=web",
		"subdomain=my-app",
		"tunnel_url=https://my-app.example.com",
	} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("expected terminal log to contain %q, got %q", want, logLine)
		}
	}
}

func TestHandleTunnelEventIgnoresLateUnhealthyAfterStopped(t *testing.T) {
	cfg := clientcfg.Config{}
	cfg.SetDefaults()

	manager := NewManager(cfg, nil)
	stoppedAt := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	manager.tunnels["tun_1"] = &tunnelRuntime{
		id: "tun_1",
		status: TunnelStatus{
			ID:        "tun_1",
			Name:      "web",
			Status:    statusStopped,
			Type:      constants.Http,
			Host:      "localhost",
			Port:      3000,
			StartedAt: stoppedAt.Add(-time.Minute),
			UpdatedAt: stoppedAt,
			StoppedAt: &stoppedAt,
		},
	}

	manager.handleTunnelEvent("tun_1", tunnelclient.Event{
		Type:  tunnelclient.EventUnhealthy,
		Error: "unhealthy tunnel",
		At:    stoppedAt.Add(time.Second),
	})

	status, err := manager.GetTunnel("tun_1")
	if err != nil {
		t.Fatalf("expected tunnel, got %v", err)
	}
	if status.Status != statusStopped {
		t.Fatalf("expected stopped status, got %q", status.Status)
	}
	if status.LastError != "" {
		t.Fatalf("expected no late error after stop, got %q", status.LastError)
	}
	if len(manager.Events("tun_1")) != 0 {
		t.Fatalf("expected late unhealthy event to be ignored, got %#v", manager.Events("tun_1"))
	}
}
