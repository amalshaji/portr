package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestAdminUsersAddSendsDefaultsAndPrintsPassword(t *testing.T) {
	var received map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/users" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"team_user":{"user":{"id":2,"email":"new@example.com"},"role":"member"},"team":{"id":1,"name":"Default Team","slug":"default-team"},"password":"generated-password"}`)
	}))
	defer server.Close()

	out, err := runAdminCommand(t, server.URL, "admin", "users", "add", "new@example.com")
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if received["email"] != "new@example.com" || received["role"] != "member" {
		t.Fatalf("unexpected request body: %#v", received)
	}
	if _, ok := received["team_slug"]; ok {
		t.Fatalf("expected omitted team_slug, got %#v", received)
	}
	want := "Added new@example.com to default-team as member.\nPassword: generated-password\n"
	if out != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", out, want)
	}
}

func TestAdminUsersAddUsesTeamAndRoleAndOmitsPasswordLine(t *testing.T) {
	var received map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"team_user":{"user":{"id":2,"email":"admin@example.com"},"role":"admin"},"team":{"id":4,"name":"Engineering","slug":"engineering"}}`)
	}))
	defer server.Close()

	out, err := runAdminCommand(t, server.URL, "admin", "users", "add", "admin@example.com", "--team", "engineering", "--role", "admin")
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if received["team_slug"] != "engineering" || received["role"] != "admin" {
		t.Fatalf("unexpected request body: %#v", received)
	}
	if strings.Contains(out, "Password:") {
		t.Fatalf("expected password line to be omitted, got %q", out)
	}
	if out != "Added admin@example.com to engineering as admin.\n" {
		t.Fatalf("unexpected output %q", out)
	}
}

func TestAdminUsersAddAcceptsFlagsBeforeEmail(t *testing.T) {
	var received map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"team_user":{"user":{"email":"admin@example.com"},"role":"admin"},"team":{"slug":"engineering"}}`)
	}))
	defer server.Close()

	_, err := runAdminCommand(t, server.URL, "admin", "users", "add", "--team=engineering", "--role", "admin", "admin@example.com")
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if received["email"] != "admin@example.com" || received["team_slug"] != "engineering" || received["role"] != "admin" {
		t.Fatalf("unexpected request body: %#v", received)
	}
}

func TestParseAdminAddUserArgsRejectsMissingFlagValues(t *testing.T) {
	for _, args := range [][]string{
		{"user@example.com", "--team"},
		{"user@example.com", "--team="},
		{"user@example.com", "--team", "--role", "admin"},
		{"user@example.com", "--role"},
	} {
		if _, err := parseAdminAddUserArgs(args); err == nil {
			t.Fatalf("expected arguments %#v to fail", args)
		}
	}
}

func TestAdminUsersAddSurfacesServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"Admin access required"}`)
	}))
	defer server.Close()

	_, err := runAdminCommand(t, server.URL, "admin", "users", "add", "new@example.com")
	if err == nil || err.Error() != "Admin access required" {
		t.Fatalf("expected server error, got %v", err)
	}
}

func TestAdminUsersAddRequiresConfiguredSecret(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("server_url: example.com\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	app := &cli.App{
		Writer:   &bytes.Buffer{},
		Flags:    []cli.Flag{&cli.StringFlag{Name: "config"}},
		Commands: []*cli.Command{adminCmd()},
	}
	err := app.Run([]string{"portr", "--config", configPath, "admin", "users", "add", "new@example.com"})
	if err == nil || !strings.Contains(err.Error(), "secret_key") {
		t.Fatalf("expected missing secret_key error, got %v", err)
	}
}

func TestAdminUsersAddHelpWorksAfterEmail(t *testing.T) {
	var out bytes.Buffer
	app := &cli.App{
		Writer:   &out,
		Flags:    []cli.Flag{&cli.StringFlag{Name: "config"}},
		Commands: []*cli.Command{adminCmd()},
	}

	err := app.Run([]string{"portr", "admin", "users", "add", "user@example.com", "--help"})
	if err != nil {
		t.Fatalf("expected help to succeed after the positional email, got %v", err)
	}
	if !strings.Contains(out.String(), "Add a user to a team") {
		t.Fatalf("expected command help output, got %q", out.String())
	}
}

func runAdminCommand(t *testing.T, serverURL string, args ...string) (string, error) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configBody := fmt.Sprintf("server_url: %s\nsecret_key: test-secret\nuse_localhost: true\n", strings.TrimPrefix(serverURL, "http://"))
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out bytes.Buffer
	app := &cli.App{
		Writer:   &out,
		Flags:    []cli.Flag{&cli.StringFlag{Name: "config"}},
		Commands: []*cli.Command{adminCmd()},
	}
	argv := append([]string{"portr", "--config", configPath}, args...)
	err := app.Run(argv)
	return out.String(), err
}
