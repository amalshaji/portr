package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/websocket"
)

func openListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func serverConfig(t *testing.T, server *httptest.Server) config.Config {
	t.Helper()
	return config.Config{
		ServerUrl:    strings.TrimPrefix(server.URL, "http://"),
		SecretKey:    "sk-doctor-test",
		UseLocalHost: true,
	}
}

func TestCheckConfigPermissionsFlagsGroupReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions are not checked on windows")
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("tunnels: []\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	check := checkConfigPermissions(path)
	if check.Status != doctorWarn {
		t.Fatalf("expected a warning for 0644, got %s", check.Status)
	}
	if !strings.Contains(check.Hint, "chmod 600") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if check := checkConfigPermissions(path); check.Status != doctorPass {
		t.Fatalf("expected pass for 0600, got %s", check.Status)
	}
}

func TestCheckServerVersionReportsVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/version" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"version":"1.2.0"}`))
	}))
	defer server.Close()

	version, check := checkServerVersion(context.Background(), serverConfig(t, server))
	if check.Status != doctorPass {
		t.Fatalf("expected pass, got %s (%s)", check.Status, check.Detail)
	}
	if version != "1.2.0" {
		t.Fatalf("expected version 1.2.0, got %q", version)
	}
	if !strings.Contains(check.Detail, "1.2.0") {
		t.Fatalf("unexpected detail %q", check.Detail)
	}
}

func TestCheckServerVersionFailsWhenUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cfg := serverConfig(t, server)
	server.Close()

	_, check := checkServerVersion(context.Background(), cfg)
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Hint, "server_url") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}
}

func TestCheckServerVersionFailsOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, check := checkServerVersion(context.Background(), serverConfig(t, server))
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Detail, "404") {
		t.Fatalf("unexpected detail %q", check.Detail)
	}
}

func TestCheckServerVersionWarnsOnSchemeInServerURL(t *testing.T) {
	// GetAdminAddress prepends a scheme, so a scheme here yields https://https://host.
	cfg := config.Config{ServerUrl: "http://localhost:8000", UseLocalHost: true}

	_, check := checkServerVersion(context.Background(), cfg)
	if check.Status != doctorWarn {
		t.Fatalf("expected warn, got %s", check.Status)
	}
	if !strings.Contains(check.Hint, "remove http://") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}
}

func TestCheckSecretKeyFailsWhenRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Invalid secret key"}`))
	}))
	defer server.Close()

	connectionID, check := checkSecretKey(context.Background(), serverConfig(t, server))
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if connectionID != "" {
		t.Fatalf("expected no connection id, got %q", connectionID)
	}
	if !strings.Contains(check.Hint, "portr auth set") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}
}

func TestCheckSecretKeyReservesTCPConnection(t *testing.T) {
	var seenType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		seenType, _ = payload["connection_type"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connection_id":"abc"}`))
	}))
	defer server.Close()

	connectionID, check := checkSecretKey(context.Background(), serverConfig(t, server))
	if check.Status != doctorPass {
		t.Fatalf("expected pass, got %s (%s)", check.Status, check.Detail)
	}
	if connectionID != "abc" {
		t.Fatalf("expected connection id abc, got %q", connectionID)
	}
	// A tcp reservation claims no subdomain, so doctor cannot collide with a
	// tunnel the same user is about to start.
	if seenType != string(constants.Tcp) {
		t.Fatalf("expected a tcp reservation, got %q", seenType)
	}
}

func TestCheckSecretKeyFailsWhenEmpty(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer server.Close()

	cfg := serverConfig(t, server)
	cfg.SecretKey = ""

	_, check := checkSecretKey(context.Background(), cfg)
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if called {
		t.Fatal("expected no request for an empty secret key")
	}
}

func TestCheckSSHEndpointHintsMissingPort(t *testing.T) {
	check := checkSSHEndpoint("example.com")
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Hint, "must include the port") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}
}

func TestCheckSSHEndpointPassesForOpenListener(t *testing.T) {
	listener := openListener(t)
	if check := checkSSHEndpoint(listener.Addr().String()); check.Status != doctorPass {
		t.Fatalf("expected pass, got %s (%s)", check.Status, check.Detail)
	}
}

func TestCheckLocalServiceWarnsWhenClosed(t *testing.T) {
	listener := openListener(t)
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	tunnel := config.Tunnel{Name: "api", Type: constants.Http, Host: host}
	fmt.Sscanf(port, "%d", &tunnel.Port)

	check, ok := checkLocalService(tunnel)
	if !ok || check.Status != doctorPass {
		t.Fatalf("expected pass while listening, got %s", check.Status)
	}

	_ = listener.Close()
	check, _ = checkLocalService(tunnel)
	if check.Status != doctorWarn {
		t.Fatalf("expected warn once closed, got %s", check.Status)
	}
	if !strings.Contains(check.Hint, "503") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}
}

func TestCheckLocalServiceHintsMissingPort(t *testing.T) {
	check, ok := checkLocalService(config.Tunnel{Name: "api", Type: constants.Http, Host: "localhost"})
	if !ok {
		t.Fatal("expected a check for an http tunnel")
	}
	if !strings.Contains(check.Hint, "set 'port:' for tunnel api") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}
}

func TestCheckLocalServiceSkipsStubTunnels(t *testing.T) {
	if _, ok := checkLocalService(config.Tunnel{Type: constants.Stub, Subdomain: "yaml"}); ok {
		t.Fatal("stub tunnels have no local port to probe")
	}
}

func TestCheckDashboardPortDetectsAnotherPortrInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("yes"))
	}))
	defer server.Close()

	cfg := config.Config{DashboardPort: dashboardPortOf(t, server)}
	if check := checkDashboardPort(cfg); check.Status != doctorWarn {
		t.Fatalf("expected warn, got %s (%s)", check.Status, check.Detail)
	}
}

func TestCheckDashboardPortDetectsForeignListener(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := config.Config{DashboardPort: dashboardPortOf(t, server)}
	check := checkDashboardPort(cfg)
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Hint, "dashboard_port") {
		t.Fatalf("unexpected hint %q", check.Hint)
	}
}

func TestCheckDashboardPortSkipsWhenDisabled(t *testing.T) {
	if check := checkDashboardPort(config.Config{DisableDashboard: true}); check.Status != doctorSkipped {
		t.Fatalf("expected skipped, got %s", check.Status)
	}
}

func dashboardPortOf(t *testing.T, server *httptest.Server) int {
	t.Helper()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatalf("split server url: %v", err)
	}
	var value int
	fmt.Sscanf(port, "%d", &value)
	return value
}

func TestRedactSecretReplacesSecret(t *testing.T) {
	if got := redactSecret("auth failed for sk-abcdef", "sk-abcdef"); strings.Contains(got, "sk-abcdef") {
		t.Fatalf("secret survived redaction: %q", got)
	}
	if got := redactSecret("nothing to redact", ""); got != "nothing to redact" {
		t.Fatalf("unexpected result %q", got)
	}
	if got := redactSecret("short abc", "abc"); got != "short abc" {
		t.Fatalf("short secrets should be left alone, got %q", got)
	}
}

func TestChecksOKIgnoresWarningsAndSkips(t *testing.T) {
	checks := []doctorCheck{{Status: doctorPass}, {Status: doctorWarn}, {Status: doctorSkipped}}
	if !checksOK(checks) {
		t.Fatal("warnings and skips must not fail the run")
	}

	if checksOK(append(checks, doctorCheck{Status: doctorFail})) {
		t.Fatal("a failed check must fail the run")
	}
}

func TestSkippedCheckNamesTheBlockingReason(t *testing.T) {
	check := skippedCheck("ssh handshake", "server unreachable")
	if check.Status != doctorSkipped {
		t.Fatalf("expected skipped, got %s", check.Status)
	}
	if check.Detail != "skipped (server unreachable)" {
		t.Fatalf("unexpected detail %q", check.Detail)
	}
}

func TestRenderDoctorTextIncludesHintsAndSummary(t *testing.T) {
	var out bytes.Buffer
	report := doctorReport{
		ClientVersion: "1.2.3",
		ConfigPath:    "/tmp/config.yaml",
		Checks: []doctorCheck{
			{Name: "server", Status: doctorFail, Detail: "unreachable", Hint: "check server_url"},
		},
	}

	if err := renderDoctorText(&out, report); err != nil {
		t.Fatalf("render: %v", err)
	}
	text := out.String()
	for _, want := range []string{"❌", "→ check server_url", "1 failed"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in output:\n%s", want, text)
		}
	}
}

func TestRenderDoctorJSONOmitsSecret(t *testing.T) {
	var out bytes.Buffer
	report := doctorReport{
		OK:     false,
		Checks: []doctorCheck{{Name: "secret key", Status: doctorFail, Detail: redactSecret("rejected sk-supersecret", "sk-supersecret")}},
	}

	if err := renderDoctorJSON(&out, report); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out.String(), "sk-supersecret") {
		t.Fatalf("secret leaked into json output:\n%s", out.String())
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if _, ok := decoded["checks"]; !ok {
		t.Fatal("expected a checks key")
	}
}

func TestRunDoctorRedactsSecretsInChecks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/version") {
			_, _ = w.Write([]byte(`{"version":"1.2.0"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		// A server that echoes the key back must not leak it into the report.
		_, _ = w.Write([]byte(`{"message":"Invalid secret key sk-doctor-test-value"}`))
	}))
	defer server.Close()

	cfg := serverConfig(t, server)
	cfg.SecretKey = "sk-doctor-test-value"
	cfg.DisableDashboard = true

	report := runDoctor(context.Background(), cfg, "/tmp/config.yaml", "1.2.3")
	for _, check := range report.Checks {
		if strings.Contains(check.Detail, cfg.SecretKey) || strings.Contains(check.Hint, cfg.SecretKey) {
			t.Fatalf("secret leaked into check %q: %+v", check.Name, check)
		}
	}
}

// runDoctorCommand runs the command with cli's exiter stubbed out, since
// app.Run calls os.Exit for an ExitCoder and would kill the test process.
func runDoctorCommand(t *testing.T, configBody string, args ...string) (string, int) {
	t.Helper()

	exitCode := 0
	originalExiter := cli.OsExiter
	cli.OsExiter = func(code int) { exitCode = code }
	t.Cleanup(func() { cli.OsExiter = originalExiter })

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out bytes.Buffer
	app := &cli.App{
		Writer:   &out,
		Flags:    []cli.Flag{&cli.StringFlag{Name: "config"}},
		Commands: []*cli.Command{doctorCmd()},
	}

	full := append([]string{"portr", "--config", configPath, "doctor"}, args...)
	_ = app.Run(full)
	return out.String(), exitCode
}

func TestDoctorCommandExitsNonZeroOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	serverURL := strings.TrimPrefix(server.URL, "http://")
	server.Close()

	body := fmt.Sprintf("server_url: %s\nssh_url: %s\nsecret_key: sk-doctor-test\nuse_localhost: true\ndisable_dashboard: true\ntunnels: []\n", serverURL, serverURL)
	out, exitCode := runDoctorCommand(t, body)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\n%s", exitCode, out)
	}
	if !strings.Contains(out, "❌") {
		t.Fatalf("expected a failed check in output:\n%s", out)
	}
}

func TestDoctorCommandJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/version") {
			_, _ = w.Write([]byte(`{"version":"1.2.0"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connection_id":"abc"}`))
	}))
	defer server.Close()
	serverURL := strings.TrimPrefix(server.URL, "http://")

	body := fmt.Sprintf("server_url: %s\nssh_url: %s\nsecret_key: sk-doctor-test\nuse_localhost: true\ndisable_dashboard: true\ntunnels: []\n", serverURL, serverURL)
	out, _ := runDoctorCommand(t, body, "--json")

	var report doctorReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, out)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected checks in the json report")
	}
	if report.ServerVersion != "1.2.0" {
		t.Fatalf("expected the server version in the report, got %q", report.ServerVersion)
	}
}

func TestDoctorCommandFailsWhenConfigMissing(t *testing.T) {
	var out bytes.Buffer
	app := &cli.App{
		Writer:   &out,
		Flags:    []cli.Flag{&cli.StringFlag{Name: "config"}},
		Commands: []*cli.Command{doctorCmd()},
	}

	err := app.Run([]string{"portr", "--config", "/nonexistent/portr/config.yaml", "doctor"})
	if err == nil {
		t.Fatal("expected an error for a missing config")
	}
	if !strings.Contains(err.Error(), "/nonexistent/portr/config.yaml") {
		t.Fatalf("expected the path in the error, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no report when the config cannot load, got %q", out.String())
	}
}

func TestRunDoctorChecksSkipsDownstreamWithTheRealReason(t *testing.T) {
	// A dead server must not make the handshake report "secret key not accepted".
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cfg := serverConfig(t, server)
	cfg.DisableDashboard = true
	server.Close()

	_, checks := runDoctorChecks(context.Background(), cfg, "/tmp/config.yaml")

	byName := map[string]doctorCheck{}
	for _, check := range checks {
		byName[check.Name] = check
	}

	for _, name := range []string{"secret key", "ssh handshake"} {
		check := byName[name]
		if check.Status != doctorSkipped {
			t.Fatalf("expected %q to be skipped, got %s", name, check.Status)
		}
		if !strings.Contains(check.Detail, "server unreachable") {
			t.Fatalf("expected %q to name the real reason, got %q", name, check.Detail)
		}
	}
}

func websocketEndpointConfig(server *httptest.Server) config.Config {
	host := strings.TrimPrefix(server.URL, "http://")
	return config.Config{
		ServerUrl:    host,
		WsUrl:        host,
		Transport:    config.TransportWebSocket,
		SecretKey:    "sk-doctor-test",
		UseLocalHost: true,
	}
}

func TestCheckWebSocketEndpointPassesOnCredentialChallenge(t *testing.T) {
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: "missing connection credentials"})
	}))
	defer server.Close()

	check := checkWebSocketEndpoint(context.Background(), websocketEndpointConfig(server))
	if check.Status != doctorPass {
		t.Fatalf("expected pass, got %s (%s)", check.Status, check.Detail)
	}
}

func TestCheckWebSocketEndpointFailsOnProtocolMismatch(t *testing.T) {
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: "portr websocket protocol mismatch: server speaks v2, client sent \"1\"; upgrade the portr client"})
	}))
	defer server.Close()

	check := checkWebSocketEndpoint(context.Background(), websocketEndpointConfig(server))
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s (%s)", check.Status, check.Detail)
	}
	if !strings.Contains(check.Detail, "protocol mismatch") {
		t.Fatalf("expected the server's mismatch message, got %q", check.Detail)
	}
}

func TestCheckWebSocketEndpointFailsWhenUnreachable(t *testing.T) {
	listener := openListener(t)
	address := listener.Addr().String()
	_ = listener.Close()

	check := checkWebSocketEndpoint(context.Background(), config.Config{
		ServerUrl:    address,
		WsUrl:        address,
		Transport:    config.TransportWebSocket,
		SecretKey:    "sk-doctor-test",
		UseLocalHost: true,
	})
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s (%s)", check.Status, check.Detail)
	}
}

func TestCheckWebSocketEndpointFailsOnUnexpectedFrame(t *testing.T) {
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		// A non-portr websocket service completes the handshake but does not
		// answer with the anonymous-connect credential challenge.
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: "welcome", Message: "hello"})
	}))
	defer server.Close()

	check := checkWebSocketEndpoint(context.Background(), websocketEndpointConfig(server))
	if check.Status != doctorFail {
		t.Fatalf("expected fail, got %s (%s)", check.Status, check.Detail)
	}
	if !strings.Contains(check.Detail, "unexpected") {
		t.Fatalf("expected an unexpected-frame detail, got %q", check.Detail)
	}
}
