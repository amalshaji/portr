package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDockerConfigIgnoresConfigFileTunnels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := `server_url: tunnel.example.com
secret_key: key
tunnels:
  - name: docs
    type: static
    dir: ./does-not-exist
groups:
  site: [docs]
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := dockerConfig(path)
	if err != nil {
		t.Fatalf("expected config-file tunnels to be ignored, got %v", err)
	}
	if len(cfg.Tunnels) != 0 || len(cfg.Groups) != 0 {
		t.Fatalf("expected no tunnels or groups, got %+v %+v", cfg.Tunnels, cfg.Groups)
	}
	if !cfg.DisableTUI {
		t.Fatal("docker mode must run without the TUI")
	}
	if cfg.ServerUrl != "tunnel.example.com" || cfg.SecretKey != "key" {
		t.Fatalf("connection settings lost: %+v", cfg)
	}
}

func TestDockerConfigValidatesSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("transport: carrier-pigeon\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := dockerConfig(path); err == nil {
		t.Fatal("expected an invalid transport to fail")
	}
}
