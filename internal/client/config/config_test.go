package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/constants"
)

func useDefaultConfigPath(t *testing.T, path string) {
	t.Helper()

	previousPath := DefaultConfigPath
	DefaultConfigPath = path
	t.Cleanup(func() {
		DefaultConfigPath = previousPath
	})
}

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

func TestGetConfigUpdatesOnlyTokenWhenDefaultConfigExists(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	existingConfig := `server_url: existing.example.com
ssh_url: existing.example.com:2222
secret_key: old-token
tunnels:
  - name: api
    subdomain: api-dev
    port: 3000
    type: http
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	requestPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if r.URL.Path != "/api/v1/config/download" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"server_url: downloaded.example.com\nssh_url: downloaded.example.com:2222\nsecret_key: new-token\ntunnels:\n  - name: downloaded\n    subdomain: downloaded\n    port: 4321"}`)
	}))
	defer server.Close()

	if err := GetConfig("new-token", server.URL); err != nil {
		t.Fatalf("get config: %v", err)
	}
	if requestPath != "/api/v1/config/download" {
		t.Fatalf("expected config download endpoint to be called, got %q", requestPath)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	configContent := string(configBytes)

	if !strings.Contains(configContent, "secret_key: new-token") {
		t.Fatalf("expected token to be updated, got: %s", configContent)
	}
	if !strings.Contains(configContent, "server_url: existing.example.com") {
		t.Fatalf("expected existing server_url to be preserved, got: %s", configContent)
	}
	if !strings.Contains(configContent, "name: api") || !strings.Contains(configContent, "subdomain: api-dev") {
		t.Fatalf("expected existing tunnel to be preserved, got: %s", configContent)
	}
	if strings.Contains(configContent, "downloaded.example.com") || strings.Contains(configContent, "name: downloaded") {
		t.Fatalf("expected downloaded template not to overwrite existing config, got: %s", configContent)
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
