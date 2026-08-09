package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

const qrBlock = "█"

func qrModel(t *testing.T, enabled bool, tunnels ...config.Tunnel) model {
	t.Helper()

	m := model{
		tunnels:   map[string]*tunnelStatus{},
		qrEnabled: enabled,
		width:     200,
		height:    60,
	}
	for _, tunnel := range tunnels {
		key := tunnelKey(&tunnel)
		m.tunnels[key] = &tunnelStatus{
			config:       &tunnel,
			clientConfig: testClientConfig(tunnel),
			active:       1,
			poolSize:     1,
		}
	}
	return m
}

func pressR(t *testing.T, m model) model {
	t.Helper()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	return updated.(model)
}

func TestViewOmitsQRHintWhenDisabled(t *testing.T) {
	view := qrModel(t, false, testTunnel()).View()
	if strings.Contains(view, "QR") {
		t.Fatalf("expected no QR hint when disabled, got %q", view)
	}
}

func TestViewShowsQRHintWhenEnabled(t *testing.T) {
	view := qrModel(t, true, testTunnel()).View()
	if !strings.Contains(view, "r: QR code") {
		t.Fatalf("expected QR hint, got %q", view)
	}
}

func TestQRKeyIgnoredWhenDisabled(t *testing.T) {
	m := pressR(t, qrModel(t, false, testTunnel()))
	if m.showQR {
		t.Fatal("expected the key to be ignored when the feature is off")
	}
	if strings.Contains(m.View(), qrBlock) {
		t.Fatal("expected no QR panel when the feature is off")
	}
}

func TestQRKeyTogglesPanelForSingleHTTPTunnel(t *testing.T) {
	m := pressR(t, qrModel(t, true, testTunnel()))

	view := m.View()
	if !strings.Contains(view, "Scan to open https://audio-stream.go.portr.dev") {
		t.Fatalf("expected QR caption, got %q", view)
	}
	if !strings.Contains(view, qrBlock) {
		t.Fatalf("expected a rendered QR code, got %q", view)
	}

	view = pressR(t, m).View()
	if strings.Contains(view, "Scan to open") || strings.Contains(view, qrBlock) {
		t.Fatalf("expected the panel to toggle off, got %q", view)
	}
}

func TestQRUnavailableWithMultipleTunnels(t *testing.T) {
	second := testTunnel()
	second.Name = "second"
	second.Subdomain = "second"
	second.Port = 9000

	view := pressR(t, qrModel(t, true, testTunnel(), second)).View()
	if !strings.Contains(view, "QR code unavailable") {
		t.Fatalf("expected unavailable note, got %q", view)
	}
	if strings.Contains(view, qrBlock) {
		t.Fatalf("expected no QR code for multiple tunnels, got %q", view)
	}
}

func TestQRUnavailableForTcpOnlyTunnel(t *testing.T) {
	tcp := config.Tunnel{
		Name:     "postgres",
		Type:     constants.Tcp,
		Host:     "localhost",
		Port:     5432,
		PoolSize: 1,
	}

	view := pressR(t, qrModel(t, true, tcp)).View()
	if !strings.Contains(view, "QR code unavailable") {
		t.Fatalf("expected unavailable note for tcp, got %q", view)
	}
}

func TestQRPanelShrinksRequestTable(t *testing.T) {
	m := qrModel(t, true, testTunnel())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m = updated.(model)
	before := m.table.Height()

	m = pressR(t, m)
	after := m.table.Height()

	panelHeight := strings.Count(m.qrPanel, "\n") + 1
	if after >= before {
		t.Fatalf("expected the table to shrink, before=%d after=%d", before, after)
	}
	if before-after < panelHeight {
		t.Fatalf("table shrank by %d, expected at least the panel height %d", before-after, panelHeight)
	}
}

func TestQRPanelAppearsAfterTunnelConnects(t *testing.T) {
	// With no tunnels the view short-circuits before the panel renders, but the
	// toggle must still be remembered.
	m := pressR(t, qrModel(t, true))
	if !m.showQR {
		t.Fatal("expected the toggle to be remembered before any tunnel connects")
	}
	if strings.Contains(m.View(), qrBlock) {
		t.Fatalf("expected no QR code with no tunnels, got %q", m.View())
	}

	tunnel := testTunnel()
	updated, _ := m.Update(AddTunnelMsg{
		Config:       &tunnel,
		ClientConfig: testClientConfig(tunnel),
		Healthy:      true,
	})
	m = updated.(model)

	if !strings.Contains(m.View(), qrBlock) {
		t.Fatalf("expected the QR panel once a tunnel connected, got %q", m.View())
	}
}

func TestQRUnavailableWhenTerminalTooNarrow(t *testing.T) {
	m := qrModel(t, true, testTunnel())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 60})
	m = pressR(t, updated.(model))

	if !strings.Contains(m.View(), "terminal too narrow") {
		t.Fatalf("expected the narrow-terminal note, got %q", m.View())
	}
}
