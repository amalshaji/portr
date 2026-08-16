package tui

import (
	"strings"
	"testing"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
)

func TestViewRendersStatusIcons(t *testing.T) {
	tunnel := testTunnel()
	m := model{
		tunnels: map[string]*tunnelStatus{
			"8765": {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				healthy:      true,
				active:       2,
				poolSize:     2,
			},
		},
		width: 1000,
	}

	view := m.View()
	if strings.Contains(view, "�") {
		t.Fatalf("expected status output without replacement character, got %q", view)
	}
	if !strings.Contains(view, "🟢 Healthy (2/2)") {
		t.Fatalf("expected healthy icon and count, got %q", view)
	}
	if strings.Contains(view, "🔒") {
		t.Fatalf("expected no auth indicator without basic auth, got %q", view)
	}
}

func TestViewRendersAuthIndicator(t *testing.T) {
	tunnel := testTunnel()
	tunnel.BasicAuth = "admin:admin"
	m := model{
		tunnels: map[string]*tunnelStatus{
			"8765": {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				active:       2,
				poolSize:     2,
			},
		},
		// Default terminal width: the right end of the line truncates,
		// so the marker must survive next to the tunnel name.
		width: 80,
	}

	view := m.View()
	if !strings.Contains(view, "audio-stream 🔒 (") {
		t.Fatalf("expected auth indicator next to tunnel name, got %q", view)
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
		width: 200,
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

func TestViewRendersStubTunnelWithoutLocalPort(t *testing.T) {
	tunnel := config.Tunnel{
		Name:      "yaml",
		Type:      constants.Stub,
		Subdomain: "yaml",
		PoolSize:  1,
	}
	m := model{
		tunnels: map[string]*tunnelStatus{
			"stub:yaml": {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				healthy:      true,
				active:       1,
				poolSize:     1,
			},
		},
		width: 200,
	}

	view := m.View()
	if strings.Contains(view, "localhost:0") {
		t.Fatalf("expected stub tunnel view without local port, got %q", view)
	}
	if !strings.Contains(view, "yaml (stub → https://yaml.go.portr.dev)") {
		t.Fatalf("expected stub tunnel address, got %q", view)
	}
}

func TestUpdateHealthFalsePreservesActivePool(t *testing.T) {
	tunnel := testTunnel()
	m := model{
		tunnels: map[string]*tunnelStatus{
			"8765": {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				healthy:      true,
				active:       2,
				poolSize:     2,
			},
		},
		width: 1000,
	}

	updated, _ := m.Update(UpdateHealthMsg{Port: "8765", Healthy: false})
	m = updated.(model)

	status := m.tunnels["8765"]
	if !status.healthy {
		t.Fatal("expected remaining pooled workers to keep the tunnel healthy")
	}
	if status.active != 2 {
		t.Fatalf("expected health updates not to overwrite the active pool count, got %d", status.active)
	}
	if !strings.Contains(m.View(), "🟢 Healthy (2/2)") {
		t.Fatalf("expected healthy pooled status in view, got %q", m.View())
	}
}

func TestWorkerLossLeavesRemainingPoolWorkerActive(t *testing.T) {
	tunnel := testTunnel()
	m := model{
		tunnels: map[string]*tunnelStatus{
			"8765": {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				healthy:      true,
				active:       2,
				poolSize:     2,
			},
		},
		width: 1000,
	}

	updated, _ := m.Update(UpdateConnCountMsg{Port: "8765", Delta: -1})
	m = updated.(model)
	updated, _ = m.Update(UpdateHealthMsg{Port: "8765", Healthy: false})
	m = updated.(model)

	status := m.tunnels["8765"]
	if !status.healthy || status.active != 1 {
		t.Fatalf("expected one healthy pooled worker, got healthy=%v active=%d", status.healthy, status.active)
	}
	if !strings.Contains(m.View(), "🟡 Partial (1/2)") {
		t.Fatalf("expected partial pooled status in view, got %q", m.View())
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

func TestViewRendersStaticTunnelWithoutLocalPort(t *testing.T) {
	tunnel := config.Tunnel{
		Name:      "site",
		Type:      constants.Static,
		Subdomain: "site",
		Dir:       "/tmp/dist",
		PoolSize:  1,
	}
	m := model{
		tunnels: map[string]*tunnelStatus{
			tunnelKey(&tunnel): {
				config:       &tunnel,
				clientConfig: testClientConfig(tunnel),
				active:       1,
				poolSize:     1,
			},
		},
		width: 200,
	}

	view := m.View()
	if !strings.Contains(view, "site (static → https://site.go.portr.dev)") {
		t.Fatalf("expected static tunnel line, got %q", view)
	}
	if strings.Contains(view, "127.0.0.1:") || strings.Contains(view, "/tmp/dist") {
		t.Fatalf("expected no local address or directory in the status line, got %q", view)
	}
}

func TestTunnelKeyDistinguishesStaticTunnelsSharingLocalPort(t *testing.T) {
	// Every static tunnel is served from one responder port, so a port-based
	// key would collapse them into a single row with a wrong health count.
	first := config.Tunnel{Type: constants.Static, Subdomain: "a", Port: 54321}
	second := config.Tunnel{Type: constants.Static, Subdomain: "b", Port: 54321}

	if tunnelKey(&first) == tunnelKey(&second) {
		t.Fatalf("expected distinct keys, both were %q", tunnelKey(&first))
	}
}
