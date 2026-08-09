package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
)

func staticDir(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(body), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	return dir
}

func TestPrepareStaticTunnelsUsesOneLocalResponder(t *testing.T) {
	c := &Client{
		config: &clientcfg.Config{
			DisableTUI: true,
		},
	}
	t.Cleanup(func() {
		c.Shutdown(t.Context())
	})

	prepared, err := c.prepareStaticTunnels([]clientcfg.ClientConfig{
		{
			Tunnel: clientcfg.Tunnel{
				Type:      constants.Static,
				Subdomain: "a",
				Dir:       staticDir(t, "A"),
			},
		},
		{
			Tunnel: clientcfg.Tunnel{
				Type:      constants.Static,
				Subdomain: "b",
				Dir:       staticDir(t, "B"),
			},
		},
	})
	if err != nil {
		t.Fatalf("prepare static tunnels: %v", err)
	}

	if len(prepared) != 2 {
		t.Fatalf("expected 2 prepared configs, got %d", len(prepared))
	}
	firstPort := prepared[0].Tunnel.Port
	if firstPort == 0 {
		t.Fatal("expected first static tunnel to receive responder port")
	}
	for _, cfg := range prepared {
		if cfg.Tunnel.Type != constants.Static {
			t.Fatalf("expected public tunnel type to remain static, got %s", cfg.Tunnel.Type)
		}
		if cfg.Tunnel.Host != "127.0.0.1" {
			t.Fatalf("expected local responder host, got %q", cfg.Tunnel.Host)
		}
		if cfg.Tunnel.Port != firstPort {
			t.Fatalf("expected shared responder port %d, got %d", firstPort, cfg.Tunnel.Port)
		}
	}

	// Every static tunnel shares one port, so Host is the only discriminator.
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/", firstPort), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "b.example.test"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request file responder: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != "B" {
		t.Fatalf("unexpected response body: %q", string(body))
	}
}
