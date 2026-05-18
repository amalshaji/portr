package tui

import (
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func TestViewRendersStatusIcons(t *testing.T) {
	tunnel := testTunnel()
	m := model{
		tunnels: map[string]*tunnelStatus{
			"8765": {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				active:       2,
				poolSize:     2,
			},
		},
		width: 200,
	}

	view := m.View()
	if strings.Contains(view, "�") {
		t.Fatalf("expected status output without replacement character, got %q", view)
	}
	if !strings.Contains(view, "🟢 Healthy (2/2)") {
		t.Fatalf("expected healthy icon and count, got %q", view)
	}
}

func TestUpdateConnCountClampsToPoolSize(t *testing.T) {
	tunnel := testTunnel()
	m := model{
		tunnels: map[string]*tunnelStatus{
			"8765": {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				poolSize:     2,
			},
		},
		width: 100,
	}

	for i := 0; i < 598; i++ {
		updated, _ := m.Update(UpdateConnCountMsg{Port: "8765", Delta: 1})
		m = updated.(model)
	}

	if got := m.tunnels["8765"].active; got != 2 {
		t.Fatalf("expected active count to clamp to pool size 2, got %d", got)
	}

	updated, _ := m.Update(UpdateConnCountMsg{Port: "8765", Delta: -5})
	m = updated.(model)
	if got := m.tunnels["8765"].active; got != 0 {
		t.Fatalf("expected active count not to drop below 0, got %d", got)
	}
}

func testTunnel() config.Tunnel {
	return config.Tunnel{
		Name:      "audio-stream",
		Type:      constants.Http,
		Host:      "localhost",
		Port:      8765,
		Subdomain: "audio-stream",
		PoolSize:  2,
	}
}

func testClientConfig(tunnel config.Tunnel) *config.ClientConfig {
	return &config.ClientConfig{
		Tunnel:       tunnel,
		TunnelUrl:    "go.portr.dev",
		UseLocalHost: false,
	}
}
