package tui

import (
	"bytes"
	"strings"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/charmbracelet/lipgloss"
	"github.com/mdp/qrterminal/v3"
)

// buildQRPanel renders the QR panel for the current terminal size, or an empty
// string when the panel is hidden.
func (m *model) buildQRPanel() string {
	if !m.qrEnabled || !m.showQR {
		return ""
	}

	url, ok := m.singleBrowsableTunnelURL()
	if !ok {
		return subtitleStyle.Render("QR code unavailable: needs exactly one http tunnel")
	}

	var buf bytes.Buffer
	qrterminal.GenerateHalfBlock(url, qrterminal.L, &buf)
	code := strings.TrimRight(buf.String(), "\n")

	// A QR wider than the terminal soft-wraps into something unscannable.
	if lipgloss.Width(code) > m.width {
		return subtitleStyle.Render("QR code unavailable: terminal too narrow")
	}

	return subtitleStyle.Render("Scan to open "+url) + "\n" + code
}

// singleBrowsableTunnelURL returns the public URL when exactly one tunnel has
// one. TCP tunnels are excluded because their address is not browsable.
func (m *model) singleBrowsableTunnelURL() (string, bool) {
	url := ""
	for _, tunnel := range m.tunnels {
		if tunnel.config == nil || tunnel.clientConfig == nil {
			continue
		}
		if tunnel.config.Type != constants.Http && tunnel.config.Type != constants.Stub {
			continue
		}
		if url != "" {
			return "", false
		}
		url = tunnel.clientConfig.GetTunnelAddr()
	}
	return url, url != ""
}
