package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/constants"
)

func TestSetDefaultsAppliesDashboardPort(t *testing.T) {
	cfg := Config{}

	cfg.SetDefaults()

	if cfg.DashboardPort != DefaultDashboardPort {
		t.Fatalf("expected dashboard port %d, got %d", DefaultDashboardPort, cfg.DashboardPort)
	}
}

func TestSetDefaultsEnablesRequestLoggingByDefault(t *testing.T) {
	cfg := Config{}

	cfg.SetDefaults()

	if cfg.EnableRequestLogging == nil {
		t.Fatal("expected request logging default to be set")
	}
	if !*cfg.EnableRequestLogging {
		t.Fatal("expected request logging to default to true")
	}
}

func TestLoadPreservesExplicitRequestLoggingFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("enable_request_logging: false\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.EnableRequestLogging == nil {
		t.Fatal("expected request logging value to be set")
	}
	if *cfg.EnableRequestLogging {
		t.Fatal("expected explicit request logging false to be preserved")
	}
}

func TestGetDashboardDisableLabel(t *testing.T) {
	cfg := Config{
		DisableDashboard: true,
	}

	if got := cfg.GetDashboardDisableLabel(); got != "disabled via config" {
		t.Fatalf("expected disabled via config, got %q", got)
	}
}

func TestValidateRejectsInvalidDashboardPortWhenEnabled(t *testing.T) {
	cfg := Config{
		DashboardPort: 70000,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestValidateAllowsInvalidDashboardPortWhenDashboardDisabled(t *testing.T) {
	cfg := Config{
		DashboardPort:    70000,
		DisableDashboard: true,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestGetDashboardAddress(t *testing.T) {
	cfg := Config{
		DashboardPort: 8888,
	}

	if got := cfg.GetDashboardAddress(); got != "http://localhost:8888" {
		t.Fatalf("expected dashboard address http://localhost:8888, got %q", got)
	}

	cfg.DisableDashboard = true
	if got := cfg.GetDashboardAddress(); got != "" {
		t.Fatalf("expected disabled dashboard address to be empty, got %q", got)
	}
}

func TestLoadResolvesStubTemplateFileRelativeToConfig(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "response.yml")
	if err := os.WriteFile(templatePath, []byte("message: {{message}}\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	configContent := `tunnels:
  - name: yaml
    type: stub
    subdomain: yaml
    response_format: application/yml
    response_tmpl_file: response.yml
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(cfg.Tunnels))
	}
	tunnel := cfg.Tunnels[0]
	if tunnel.Type != constants.Stub {
		t.Fatalf("expected stub tunnel, got %s", tunnel.Type)
	}
	if tunnel.ResponseTemplate != "message: {{message}}\n" {
		t.Fatalf("unexpected response template: %q", tunnel.ResponseTemplate)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid stub config, got %v", err)
	}
}

func TestLoadRejectsStubTunnelWithoutTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	configContent := `tunnels:
  - name: yaml
    type: stub
    subdomain: yaml
    response_format: application/yml
`
	if err := os.WriteFile(path, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected missing template error")
	}
	if !strings.Contains(err.Error(), "response_tmpl or response_tmpl_file is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsStubTunnelWithBothTemplateSources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "response.yml"), []byte("message: file\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	path := filepath.Join(dir, "config.yaml")
	configContent := `tunnels:
  - name: yaml
    type: stub
    subdomain: yaml
    response_format: application/yml
    response_tmpl: "message: inline"
    response_tmpl_file: response.yml
`
	if err := os.WriteFile(path, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected both template sources error")
	}
	if !strings.Contains(err.Error(), "only one of response_tmpl or response_tmpl_file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsStubTunnelWithoutResponseFormat(t *testing.T) {
	cfg := Config{
		Tunnels: []Tunnel{{
			Type:             constants.Stub,
			Subdomain:        "yaml",
			ResponseTemplate: "message: {{message}}",
		}},
	}
	cfg.SetDefaults()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected missing response format error")
	}
	if !strings.Contains(err.Error(), "response_format is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsStubTunnelWithoutSubdomain(t *testing.T) {
	cfg := Config{
		Tunnels: []Tunnel{{
			Type:             constants.Stub,
			ResponseFormat:   "application/json",
			ResponseTemplate: "{}",
		}},
	}
	cfg.SetDefaults()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected missing subdomain error")
	}
	if !strings.Contains(err.Error(), "subdomain is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
