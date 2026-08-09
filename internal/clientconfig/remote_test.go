package clientconfig

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigUpdatesAuthValuesWhenDefaultConfigExists(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	existingConfig := `server_url: existing.example.com
transport: ssh
ssh_url: existing.example.com:2222
ws_url: existing.example.com:8001
tunnel_url: existing.example.com
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
		fmt.Fprint(w, `{"message":"server_url: admin.example.com\ntransport: websocket\nws_url: tunnel.example.com\nsecret_key: new-token\ntunnels:\n  - name: downloaded\n    subdomain: downloaded\n    port: 4321"}`)
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
	if !strings.Contains(configContent, "server_url: admin.example.com") {
		t.Fatalf("expected server_url to be updated, got: %s", configContent)
	}
	if !strings.Contains(configContent, "transport: websocket") {
		t.Fatalf("expected transport to be updated, got: %s", configContent)
	}
	if !strings.Contains(configContent, "ws_url: tunnel.example.com") {
		t.Fatalf("expected ws_url to be updated, got: %s", configContent)
	}
	if !strings.Contains(configContent, "tunnel_url: admin.example.com") {
		t.Fatalf("expected legacy websocket config to fall back to server_url, got: %s", configContent)
	}
	if !strings.Contains(configContent, "ssh_url: existing.example.com:2222") {
		t.Fatalf("expected existing ssh_url to be preserved when websocket template omits it, got: %s", configContent)
	}
	if !strings.Contains(configContent, "name: api") || !strings.Contains(configContent, "subdomain: api-dev") {
		t.Fatalf("expected existing tunnel to be preserved, got: %s", configContent)
	}
	if strings.Contains(configContent, "name: downloaded") {
		t.Fatalf("expected downloaded tunnels not to overwrite existing config, got: %s", configContent)
	}
}

func TestGetConfigUpdatesSSHAuthValuesWhenDefaultConfigExists(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	if err := os.WriteFile(configPath, []byte("secret_key: old-token\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"server_url: admin.example.com\ntransport: ssh\nssh_url: ssh.example.com:2222\ntunnel_url: public.example.com\nsecret_key: new-token"}`)
	}))
	defer server.Close()

	if err := GetConfig("new-token", server.URL); err != nil {
		t.Fatalf("get config: %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	configContent := string(configBytes)
	for _, expected := range []string{
		"secret_key: new-token",
		"server_url: admin.example.com",
		"transport: ssh",
		"ssh_url: ssh.example.com:2222",
		"tunnel_url: public.example.com",
	} {
		if !strings.Contains(configContent, expected) {
			t.Fatalf("expected config to include %q, got: %s", expected, configContent)
		}
	}
}

func templateServer(t *testing.T, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/config/download" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	return server
}

func TestPullConfigReplacesTunnelsAndGroups(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	server := templateServer(t, `{"has_template":true,"message":"server_url: downloaded.example.com\nssh_url: downloaded.example.com:2222\nsecret_key: team-token\ntunnels:\n  - name: web\n    subdomain: acme-web\n    port: 3000\n  - name: api\n    subdomain: acme-api\n    port: 8000\ngroups:\n  frontend: [web, api]\n"}`)

	existingConfig := `# my portr config
server_url: ` + server.URL + `
secret_key: local-token
disable_tui: true
tunnels:
  - name: legacy
    subdomain: legacy
    port: 9000
groups:
  legacy-group: [legacy]
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := PullConfig(); err != nil {
		t.Fatalf("pull config: %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	configContent := string(configBytes)

	for _, want := range []string{"name: web", "subdomain: acme-api", "frontend:"} {
		if !strings.Contains(configContent, want) {
			t.Fatalf("expected template content %q, got: %s", want, configContent)
		}
	}
	for _, unwanted := range []string{"name: legacy", "legacy-group"} {
		if strings.Contains(configContent, unwanted) {
			t.Fatalf("expected %q to be replaced, got: %s", unwanted, configContent)
		}
	}
	if !strings.Contains(configContent, "secret_key: local-token") {
		t.Fatalf("expected secret key to be untouched, got: %s", configContent)
	}
	if !strings.Contains(configContent, "server_url: "+server.URL) {
		t.Fatalf("expected server_url to be untouched, got: %s", configContent)
	}
	if !strings.Contains(configContent, "disable_tui: true") {
		t.Fatalf("expected unrelated settings to be kept, got: %s", configContent)
	}
	if !strings.Contains(configContent, "# my portr config") {
		t.Fatalf("expected comments to be kept, got: %s", configContent)
	}
}

func TestPullConfigDropsGroupsMissingFromTemplate(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	server := templateServer(t, `{"has_template":true,"message":"secret_key: team-token\ntunnels:\n  - name: web\n    subdomain: acme-web\n    port: 3000\n"}`)

	existingConfig := `server_url: ` + server.URL + `
secret_key: local-token
tunnels:
  - name: legacy
    subdomain: legacy
    port: 9000
groups:
  legacy-group: [legacy]
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := PullConfig(); err != nil {
		t.Fatalf("pull config: %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(configBytes), "groups:") {
		t.Fatalf("expected stale groups to be dropped, got: %s", configBytes)
	}
}

func TestPullConfigWithoutTeamTemplateLeavesConfigAlone(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	useDefaultConfigPath(t, configPath)

	server := templateServer(t, `{"has_template":false,"message":"secret_key: team-token\ntunnels:\n  - name: portr\n    subdomain: portr\n    port: 4321\n"}`)

	existingConfig := `server_url: ` + server.URL + `
secret_key: local-token
tunnels:
  - name: legacy
    subdomain: legacy
    port: 9000
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := PullConfig()
	if err == nil {
		t.Fatal("expected pull to fail without a team template")
	}
	if !strings.Contains(err.Error(), "no client template configured") {
		t.Fatalf("unexpected error: %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(configBytes) != existingConfig {
		t.Fatalf("expected config to be untouched, got: %s", configBytes)
	}
}

func TestPullConfigWithoutConfigFile(t *testing.T) {
	useDefaultConfigPath(t, filepath.Join(t.TempDir(), "config.yaml"))

	_, err := PullConfig()
	if err == nil {
		t.Fatal("expected pull to fail without a config file")
	}
	if !strings.Contains(err.Error(), "portr auth set") {
		t.Fatalf("unexpected error: %v", err)
	}
}
