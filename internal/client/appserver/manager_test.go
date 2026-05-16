package appserver

import (
	"bytes"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	sshclient "github.com/amalshaji/portr/internal/client/ssh"
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

func TestClientConfigForTunnelDisablesSSHClientTerminalLogs(t *testing.T) {
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

	manager.recordEvent(tunnel, sshclient.Event{
		Type: sshclient.EventStarted,
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

func TestHandleSSHEventIgnoresLateUnhealthyAfterStopped(t *testing.T) {
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

	manager.handleSSHEvent("tun_1", sshclient.Event{
		Type:  sshclient.EventUnhealthy,
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
