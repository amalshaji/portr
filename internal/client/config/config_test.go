package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
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

func TestSetDefaultsNormalizesSubdomain(t *testing.T) {
	cfg := Config{Tunnels: []Tunnel{{Type: constants.Http, Subdomain: "  My-App  "}}}

	cfg.SetDefaults()

	if got := cfg.Tunnels[0].Subdomain; got != "my-app" {
		t.Fatalf("expected normalized subdomain, got %q", got)
	}
}

func TestValidateRejectsUnderscoreSubdomain(t *testing.T) {
	cfg := Config{Tunnels: []Tunnel{{Type: constants.Http, Subdomain: "my_app"}}}
	cfg.SetDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected underscore subdomain to be rejected")
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

func TestSetDefaultsAppliesDefaultRedactHeaders(t *testing.T) {
	cfg := Config{}

	cfg.SetDefaults()

	if len(cfg.RedactHeaders) != len(DefaultRedactHeaders) {
		t.Fatalf("expected %d redact headers, got %d", len(DefaultRedactHeaders), len(cfg.RedactHeaders))
	}
	for i, want := range DefaultRedactHeaders {
		if cfg.RedactHeaders[i] != want {
			t.Fatalf("expected redact header %q at index %d, got %q", want, i, cfg.RedactHeaders[i])
		}
	}
}

func TestLoadPreservesExplicitRedactHeaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := "redact_headers:\n  - X-Test-Secret\n  - X-Another-Secret\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.RedactHeaders) != 2 || cfg.RedactHeaders[0] != "X-Test-Secret" || cfg.RedactHeaders[1] != "X-Another-Secret" {
		t.Fatalf("expected explicit redact headers to be preserved, got %#v", cfg.RedactHeaders)
	}
}

func TestLoadAcceptsDeprecatedHTTPReverseProxyOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := "enable_http_reverse_proxy: false\nserver_url: example.test\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config with deprecated option: %v", err)
	}
	if cfg.ServerUrl != "example.test" {
		t.Fatalf("expected remaining config to load, got server_url=%q", cfg.ServerUrl)
	}
}

func TestGetConfigUpdatesAuthValuesWhenDefaultConfigExists(t *testing.T) {
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
	if !strings.Contains(configContent, "server_url: downloaded.example.com") {
		t.Fatalf("expected server_url to be updated, got: %s", configContent)
	}
	if !strings.Contains(configContent, "ssh_url: downloaded.example.com:2222") {
		t.Fatalf("expected ssh_url to be updated, got: %s", configContent)
	}
	if !strings.Contains(configContent, "tunnel_url: downloaded.example.com") {
		t.Fatalf("expected tunnel_url to be updated, got: %s", configContent)
	}
	if !strings.Contains(configContent, "name: api") || !strings.Contains(configContent, "subdomain: api-dev") {
		t.Fatalf("expected existing tunnel to be preserved, got: %s", configContent)
	}
	if strings.Contains(configContent, "name: downloaded") {
		t.Fatalf("expected downloaded tunnels not to overwrite existing config, got: %s", configContent)
	}
}

func TestUpdateConfigValuesPopulatesEmptyConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	entries := [][2]string{{"secret_key", "tok"}, {"server_url", "s.example.com"}}
	if err := updateConfigValues(entries); err != nil {
		t.Fatalf("update config values: %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	configContent := string(configBytes)

	if !strings.Contains(configContent, "secret_key: tok") || !strings.Contains(configContent, "server_url: s.example.com") {
		t.Fatalf("expected values to be written to empty config, got: %s", configContent)
	}
}

func assertPrivateFileMode(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected private file permissions, got %v", info.Mode().Perm())
	}
}

func TestSetConfigWritesPrivateConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	if err := SetConfig("secret_key: dummy-token\n"); err != nil {
		t.Fatalf("set config: %v", err)
	}

	assertPrivateFileMode(t, configPath)
}

func TestUpdateConfigValuesCorrectsExistingConfigFilePermissions(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	if err := os.WriteFile(configPath, []byte("secret_key: old-token\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Chmod(configPath, 0o644); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	if err := updateConfigValues([][2]string{{"secret_key", "new-token"}}); err != nil {
		t.Fatalf("update config values: %v", err)
	}

	assertPrivateFileMode(t, configPath)
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

func groupedTunnelsConfig() Config {
	return Config{
		Tunnels: []Tunnel{
			{Name: "web", Port: 3000},
			{Name: "api", Port: 8000},
			{Name: "pg", Type: constants.Tcp, Port: 5432},
		},
		Groups: map[string][]string{"frontend": {"web", "api"}},
	}
}

func selectedTunnelNames(tunnels []Tunnel) []string {
	names := make([]string, 0, len(tunnels))
	for _, tunnel := range tunnels {
		names = append(names, tunnel.Name)
	}
	return names
}

func TestSelectTunnelsExpandsGroup(t *testing.T) {
	cfg := groupedTunnelsConfig()

	got := selectedTunnelNames(cfg.SelectTunnels([]string{"frontend"}))

	want := []string{"web", "api"}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSelectTunnelsCombinesGroupAndTunnelNames(t *testing.T) {
	cfg := groupedTunnelsConfig()

	got := selectedTunnelNames(cfg.SelectTunnels([]string{"frontend", "pg"}))

	want := []string{"web", "api", "pg"}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSelectTunnelsSelectsOverlappingTunnelOnce(t *testing.T) {
	cfg := groupedTunnelsConfig()

	got := selectedTunnelNames(cfg.SelectTunnels([]string{"frontend", "api"}))

	want := []string{"web", "api"}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSelectTunnelsWithoutServicesSelectsAll(t *testing.T) {
	cfg := groupedTunnelsConfig()

	got := selectedTunnelNames(cfg.SelectTunnels(nil))

	want := []string{"web", "api", "pg"}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSelectTunnelsWithUnknownNameSelectsNothing(t *testing.T) {
	cfg := groupedTunnelsConfig()

	if got := cfg.SelectTunnels([]string{"frontnd"}); len(got) != 0 {
		t.Fatalf("expected no tunnels, got %q", selectedTunnelNames(got))
	}
}

func TestReplaceTunnelsDropsGroups(t *testing.T) {
	cfg := groupedTunnelsConfig()

	cfg.ReplaceTunnels(Tunnel{Name: "cli", Port: 4000})
	cfg.SetDefaults()

	if cfg.Groups != nil {
		t.Fatalf("expected groups to be dropped, got %v", cfg.Groups)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config after replacing tunnels, got %v", err)
	}
}

func TestValidateRejectsGroupWithUnknownTunnel(t *testing.T) {
	cfg := Config{
		Tunnels: []Tunnel{{Name: "web", Port: 3000}},
		Groups:  map[string][]string{"frontend": {"web", "missing"}},
	}
	cfg.SetDefaults()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected unknown tunnel error")
	}
	if !strings.Contains(err.Error(), "references unknown tunnel") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsGroupNameMatchingTunnelName(t *testing.T) {
	cfg := Config{
		Tunnels: []Tunnel{{Name: "web", Port: 3000}},
		Groups:  map[string][]string{"web": {"web"}},
	}
	cfg.SetDefaults()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected group name conflict error")
	}
	if !strings.Contains(err.Error(), "conflicts with a tunnel") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsGroupMemberWithEmptyName(t *testing.T) {
	cfg := Config{
		Tunnels: []Tunnel{{Port: 3000}, {Name: "api", Port: 8000}},
		Groups:  map[string][]string{"frontend": {""}},
	}
	cfg.SetDefaults()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected unnamed tunnel to be unreferenceable")
	}
	if !strings.Contains(err.Error(), "references unknown tunnel") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsEmptyGroup(t *testing.T) {
	cfg := Config{
		Tunnels: []Tunnel{{Name: "web", Port: 3000}},
		Groups:  map[string][]string{"frontend": {}},
	}
	cfg.SetDefaults()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected empty group error")
	}
	if !strings.Contains(err.Error(), "must reference at least one tunnel") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadParsesTunnelGroups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	configContent := `tunnels:
  - name: web
    port: 3000
  - name: api
    port: 8000
groups:
  frontend: [web, api]
`
	if err := os.WriteFile(path, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	want := []string{"web", "api"}
	if !slices.Equal(cfg.Groups["frontend"], want) {
		t.Fatalf("expected %q, got %q", want, cfg.Groups["frontend"])
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestNoMatchingTunnelsErrorListsTunnelsAndGroups(t *testing.T) {
	cfg := Config{
		Tunnels: []Tunnel{{Name: "web", Port: 3000}, {Name: "api", Port: 8000}},
		Groups:  map[string][]string{"frontend": {"web", "api"}},
	}

	err := cfg.NoMatchingTunnelsError([]string{"frontnd"})
	if err == nil {
		t.Fatal("expected no matching tunnels error")
	}
	if !strings.Contains(err.Error(), "frontnd") {
		t.Fatalf("expected error to name the unknown service, got %v", err)
	}
	if !strings.Contains(err.Error(), "available: api, frontend (group), web") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNoMatchingTunnelsErrorWithoutNamedTunnels(t *testing.T) {
	cfg := Config{Tunnels: []Tunnel{{Port: 3000}}}

	err := cfg.NoMatchingTunnelsError([]string{"web"})
	if err == nil {
		t.Fatal("expected no named tunnels error")
	}
	if !strings.Contains(err.Error(), "no named tunnels configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvedHostHeaderRewriteUsesLocalAddr(t *testing.T) {
	for _, value := range []string{"rewrite", "  Rewrite  ", "REWRITE"} {
		tunnel := Tunnel{HostHeader: value}
		if got := tunnel.ResolvedHostHeader("localhost:3000"); got != "localhost:3000" {
			t.Fatalf("host_header %q resolved to %q", value, got)
		}
	}
}

func TestResolvedHostHeaderReturnsLiteralValue(t *testing.T) {
	tunnel := Tunnel{HostHeader: "myapp.local"}
	if got := tunnel.ResolvedHostHeader("localhost:3000"); got != "myapp.local" {
		t.Fatalf("expected literal host header, got %q", got)
	}
}

func TestResolvedHostHeaderEmptyPassesThrough(t *testing.T) {
	tunnel := Tunnel{}
	if got := tunnel.ResolvedHostHeader("localhost:3000"); got != "" {
		t.Fatalf("expected pass-through, got %q", got)
	}
}

func TestValidateRejectsHostHeaderOnNonHTTPTunnel(t *testing.T) {
	tcp := Tunnel{Type: constants.Tcp, Port: 5432, HostHeader: "rewrite"}
	tcp.SetDefaults()
	err := tcp.Validate()
	if err == nil || !strings.Contains(err.Error(), "host_header is only supported for http tunnels") {
		t.Fatalf("expected tcp rejection, got %v", err)
	}

	// Stub tunnels reach the same reverse proxy but route by Host, so a rewrite
	// would silently serve the wrong stub when more than one is registered.
	stub := Tunnel{
		Type:             constants.Stub,
		Subdomain:        "stubby",
		ResponseFormat:   "application/json",
		ResponseTemplate: "{}",
		HostHeader:       "rewrite",
	}
	stub.SetDefaults()
	err = stub.Validate()
	if err == nil || !strings.Contains(err.Error(), "host_header is only supported for http tunnels") {
		t.Fatalf("expected stub rejection, got %v", err)
	}

	http := Tunnel{Type: constants.Http, Subdomain: "web", Port: 3000, HostHeader: "rewrite"}
	http.SetDefaults()
	if err := http.Validate(); err != nil {
		t.Fatalf("http tunnel should accept host_header: %v", err)
	}
}
