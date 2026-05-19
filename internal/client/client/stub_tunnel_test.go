package client

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func TestPrepareStubTunnelsUsesOneLocalResponder(t *testing.T) {
	c := &Client{
		config: &clientcfg.Config{
			DisableTUI: true,
		},
	}
	t.Cleanup(func() {
		c.Shutdown(t.Context())
	})

	prepared, err := c.prepareStubTunnels([]clientcfg.ClientConfig{
		{
			Tunnel: clientcfg.Tunnel{
				Type:             constants.Stub,
				Subdomain:        "json",
				ResponseFormat:   "application/json",
				ResponseTemplate: `{"name":"{{name}}"}`,
			},
		},
		{
			Tunnel: clientcfg.Tunnel{
				Type:             constants.Stub,
				Subdomain:        "yaml",
				ResponseFormat:   "application/yml",
				ResponseTemplate: "name: {{name}}\n",
			},
		},
	})
	if err != nil {
		t.Fatalf("prepare stub tunnels: %v", err)
	}

	if len(prepared) != 2 {
		t.Fatalf("expected 2 prepared configs, got %d", len(prepared))
	}
	firstPort := prepared[0].Tunnel.Port
	if firstPort == 0 {
		t.Fatal("expected first stub tunnel to receive responder port")
	}
	for _, cfg := range prepared {
		if cfg.Tunnel.Type != constants.Stub {
			t.Fatalf("expected public tunnel type to remain stub, got %s", cfg.Tunnel.Type)
		}
		if cfg.Tunnel.Host != "127.0.0.1" {
			t.Fatalf("expected local responder host, got %q", cfg.Tunnel.Host)
		}
		if cfg.Tunnel.Port != firstPort {
			t.Fatalf("expected shared responder port %d, got %d", firstPort, cfg.Tunnel.Port)
		}
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/?name=portr", firstPort), nil)
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
