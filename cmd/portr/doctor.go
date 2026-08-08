package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	requestlogs "github.com/amalshaji/portr/internal/client/logs"
	sshclient "github.com/amalshaji/portr/internal/client/ssh"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/urfave/cli/v2"
)

type doctorStatus string

const (
	doctorPass    doctorStatus = "pass"
	doctorWarn    doctorStatus = "warn"
	doctorFail    doctorStatus = "fail"
	doctorSkipped doctorStatus = "skipped"
)

// checksOK reports whether the run passed. Warnings and skips do not fail it.
func checksOK(checks []doctorCheck) bool {
	for _, check := range checks {
		if check.Status == doctorFail {
			return false
		}
	}
	return true
}

func skippedCheck(name, reason string) doctorCheck {
	return doctorCheck{Name: name, Status: doctorSkipped, Detail: "skipped (" + reason + ")"}
}

type doctorCheck struct {
	Name   string       `json:"name"`
	Status doctorStatus `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Hint   string       `json:"hint,omitempty"`
}

type doctorReport struct {
	OK            bool          `json:"ok"`
	ClientVersion string        `json:"client_version"`
	ConfigPath    string        `json:"config_path"`
	ServerURL     string        `json:"server_url"`
	SSHURL        string        `json:"ssh_url"`
	ServerVersion string        `json:"server_version,omitempty"`
	Checks        []doctorCheck `json:"checks"`
}

func doctorCmd() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Diagnose connection and configuration problems",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output the report as JSON",
			},
		},
		Action: func(c *cli.Context) error {
			configPath := c.String("config")

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}

			report := runDoctor(c.Context, cfg, configPath, version)

			if c.Bool("json") {
				if err := renderDoctorJSON(c.App.Writer, report); err != nil {
					return err
				}
			} else if err := renderDoctorText(c.App.Writer, report); err != nil {
				return err
			}

			if !report.OK {
				return cli.Exit("", 1)
			}
			return nil
		},
	}
}

func runDoctor(ctx context.Context, cfg config.Config, configPath, clientVersion string) doctorReport {
	report := doctorReport{
		ClientVersion: clientVersion,
		ConfigPath:    configPath,
		ServerURL:     cfg.ServerUrl,
		SSHURL:        cfg.SshUrl,
	}

	serverVersion, checks := runDoctorChecks(ctx, cfg, configPath)
	report.ServerVersion = serverVersion
	report.Checks = checks

	report.OK = checksOK(checks)

	// Details and hints can embed server errors; scrub before anyone shares this.
	for i := range report.Checks {
		report.Checks[i].Detail = redactSecret(report.Checks[i].Detail, cfg.SecretKey)
		report.Checks[i].Hint = redactSecret(report.Checks[i].Hint, cfg.SecretKey)
	}

	return report
}

func runDoctorChecks(ctx context.Context, cfg config.Config, configPath string) (string, []doctorCheck) {
	checks := []doctorCheck{
		{Name: "config file", Status: doctorPass, Detail: fmt.Sprintf("parsed, %s configured", pluralize(len(cfg.Tunnels), "tunnel"))},
		checkConfigPermissions(configPath),
		checkRequestLogDatabase(requestlogs.DefaultDBPath()),
	}

	serverVersion, serverCheck := checkServerVersion(ctx, cfg)
	checks = append(checks, serverCheck)

	// Each of these depends on the one before it. Track why the chain broke so a
	// downstream check reports that reason instead of repeating the same error.
	blocked := ""
	if serverCheck.Status == doctorFail {
		blocked = "server unreachable"
	}

	var connectionID string
	if blocked != "" {
		checks = append(checks, skippedCheck("secret key", blocked))
	} else {
		var secretCheck doctorCheck
		connectionID, secretCheck = checkSecretKey(ctx, cfg)
		checks = append(checks, secretCheck)
		if connectionID == "" {
			blocked = "secret key not accepted"
		}
	}

	endpointCheck := checkSSHEndpoint(cfg.SshUrl)
	checks = append(checks, endpointCheck)
	if blocked == "" && endpointCheck.Status == doctorFail {
		blocked = "ssh endpoint unreachable"
	}

	if blocked != "" {
		checks = append(checks, skippedCheck("ssh handshake", blocked))
	} else {
		checks = append(checks, checkSSHHandshake(ctx, cfg, connectionID))
	}

	for _, tunnel := range cfg.Tunnels {
		if check, ok := checkLocalService(tunnel); ok {
			checks = append(checks, check)
		}
	}

	checks = append(checks, checkDashboardPort(cfg))
	return serverVersion, checks
}

func checkConfigPermissions(path string) doctorCheck {
	check := doctorCheck{Name: "config permissions"}

	if runtime.GOOS == "windows" {
		check.Status = doctorSkipped
		check.Detail = "not applicable on windows"
		return check
	}

	info, err := os.Stat(path)
	if err != nil {
		check.Status = doctorFail
		check.Detail = err.Error()
		return check
	}

	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		check.Status = doctorWarn
		check.Detail = fmt.Sprintf("mode %#o is readable by other users", mode)
		check.Hint = fmt.Sprintf("chmod 600 %s", path)
		return check
	}

	check.Status = doctorPass
	check.Detail = fmt.Sprintf("mode %#o", mode)
	return check
}

func checkRequestLogDatabase(dbPath string) doctorCheck {
	check := doctorCheck{Name: "request log db"}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		check.Status = doctorPass
		check.Detail = fmt.Sprintf("%s not created yet (created on first tunnel)", dbPath)
		return check
	}

	store, err := requestlogs.Open(dbPath)
	if err != nil {
		check.Status = doctorFail
		check.Detail = err.Error()
		check.Hint = fmt.Sprintf("delete %s to recreate it (this removes local request logs)", dbPath)
		return check
	}
	_ = store.Close()

	check.Status = doctorPass
	check.Detail = dbPath
	return check
}

func checkServerVersion(ctx context.Context, cfg config.Config) (string, doctorCheck) {
	check := doctorCheck{Name: "server"}

	if strings.HasPrefix(cfg.ServerUrl, "http://") || strings.HasPrefix(cfg.ServerUrl, "https://") {
		check.Status = doctorWarn
		check.Detail = fmt.Sprintf("server_url %q includes a scheme", cfg.ServerUrl)
		check.Hint = "remove http:// or https:// from server_url; portr adds it"
		return "", check
	}

	baseURL := cfg.GetAdminAddress()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/version", nil)
	if err != nil {
		check.Status = doctorFail
		check.Detail = err.Error()
		return "", check
	}

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		check.Status = doctorFail
		check.Detail = err.Error()
		check.Hint = "check server_url; if the server runs over http, set use_localhost: true"
		return "", check
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		check.Status = doctorFail
		check.Detail = fmt.Sprintf("%s returned %d", baseURL, response.StatusCode)
		check.Hint = "server_url may point at a proxy or the wrong host"
		return "", check
	}

	var payload struct {
		Version string `json:"version"`
	}
	body, _ := io.ReadAll(response.Body)
	_ = json.Unmarshal(body, &payload)

	check.Status = doctorPass
	if payload.Version == "" {
		check.Detail = baseURL
	} else {
		check.Detail = fmt.Sprintf("%s (version %s)", baseURL, payload.Version)
	}
	return payload.Version, check
}

// checkSecretKey reserves a tcp-type connection. A tcp reservation claims no
// subdomain and no port, so it cannot collide with a tunnel the user is about
// to start, and the server reaps unclaimed reservations after five minutes.
func checkSecretKey(ctx context.Context, cfg config.Config) (string, doctorCheck) {
	check := doctorCheck{Name: "secret key"}

	if strings.TrimSpace(cfg.SecretKey) == "" {
		check.Status = doctorFail
		check.Detail = "secret_key is empty"
		check.Hint = "run: portr auth set --token <your token> --remote " + cfg.ServerUrl
		return "", check
	}

	connectionID, err := sshclient.CreateNewConnectionWithContext(ctx, doctorClientConfig(cfg))
	if err != nil {
		check.Status = doctorFail
		check.Detail = err.Error()
		check.Hint = "run: portr auth set --token <your token> --remote " + cfg.ServerUrl
		return "", check
	}

	check.Status = doctorPass
	check.Detail = fmt.Sprintf("accepted by %s (reserved a short-lived diagnostic connection)", cfg.ServerUrl)
	return connectionID, check
}

func checkSSHEndpoint(sshURL string) doctorCheck {
	check := doctorCheck{Name: "ssh endpoint"}

	if _, _, err := net.SplitHostPort(sshURL); err != nil {
		check.Status = doctorFail
		check.Detail = fmt.Sprintf("ssh_url %q is not host:port", sshURL)
		check.Hint = "ssh_url must include the port, e.g. portr.example.com:2222"
		return check
	}

	conn, err := net.DialTimeout("tcp", sshURL, 5*time.Second)
	if err != nil {
		check.Status = doctorFail
		check.Detail = err.Error()
		check.Hint = "check ssh_url and that outbound traffic to that port is not blocked"
		return check
	}
	_ = conn.Close()

	check.Status = doctorPass
	check.Detail = "reachable at " + sshURL
	return check
}

func checkSSHHandshake(ctx context.Context, cfg config.Config, connectionID string) doctorCheck {
	check := doctorCheck{Name: "ssh handshake"}

	clientConfig := doctorClientConfig(cfg)
	clientConfig.ConnectionID = connectionID

	fingerprint, err := sshclient.Probe(ctx, clientConfig)
	if err != nil {
		check.Status = doctorFail
		check.Detail = err.Error()
		check.Hint = "verify ssh_url points at the portr ssh port, not the http port"
		return check
	}

	check.Status = doctorPass
	check.Detail = "authenticated, host key " + fingerprint
	if clientConfig.InsecureSkipHostKeyVerification {
		check.Detail += " (host key verification disabled)"
	}
	return check
}

func checkLocalService(tunnel config.Tunnel) (doctorCheck, bool) {
	// Stub tunnels are served in-process; there is no local port to probe.
	if tunnel.Type == constants.Stub {
		return doctorCheck{}, false
	}

	check := doctorCheck{Name: fmt.Sprintf("local service (%s)", tunnel.DisplayName())}
	addr := tunnel.GetLocalAddr()

	if tunnel.Port <= 0 {
		check.Status = doctorWarn
		check.Detail = "no local port configured"
		check.Hint = fmt.Sprintf("set 'port:' for tunnel %s", tunnel.DisplayName())
		return check, true
	}

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		check.Status = doctorWarn
		check.Detail = fmt.Sprintf("nothing is listening on %s", addr)
		check.Hint = "requests will return 503 until it starts"
		return check, true
	}
	_ = conn.Close()

	check.Status = doctorPass
	check.Detail = "listening on " + addr
	return check, true
}

func checkDashboardPort(cfg config.Config) doctorCheck {
	check := doctorCheck{Name: "dashboard port"}

	if cfg.DisableDashboard {
		check.Status = doctorSkipped
		check.Detail = "disabled via config"
		return check
	}

	// Mirrors the probe the dashboard itself does before binding.
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://localhost:%d/is-this-portr-server", cfg.DashboardPort))
	if err != nil {
		check.Status = doctorPass
		check.Detail = fmt.Sprintf("port %d is free", cfg.DashboardPort)
		return check
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	if response.StatusCode == http.StatusOK && string(body) == "yes" {
		check.Status = doctorWarn
		check.Detail = fmt.Sprintf("another portr instance already owns port %d", cfg.DashboardPort)
		return check
	}

	check.Status = doctorFail
	check.Detail = fmt.Sprintf("port %d is in use by another application", cfg.DashboardPort)
	check.Hint = "set dashboard_port in the config"
	return check
}

func pluralize(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, noun)
	}
	return fmt.Sprintf("%d %ss", count, noun)
}

func doctorClientConfig(cfg config.Config) config.ClientConfig {
	return config.ClientConfig{
		ServerUrl:                       cfg.ServerUrl,
		SshUrl:                          cfg.SshUrl,
		TunnelUrl:                       cfg.TunnelUrl,
		SecretKey:                       cfg.SecretKey,
		UseLocalHost:                    cfg.UseLocalHost,
		Debug:                           cfg.Debug,
		InsecureSkipHostKeyVerification: cfg.InsecureSkipHostKeyVerification == nil || *cfg.InsecureSkipHostKeyVerification,
		// A tcp reservation claims no subdomain, so it never collides with a
		// tunnel the user is about to start.
		Tunnel: config.Tunnel{Type: constants.Tcp},
	}
}

// redactSecret keeps the secret key out of output that is meant to be pasted
// into bug reports. Short values are left alone so junk keys do not mangle text.
func redactSecret(text, secret string) string {
	if len(secret) < 6 {
		return text
	}
	return strings.ReplaceAll(text, secret, "***")
}

func doctorStatusIcon(status doctorStatus) string {
	switch status {
	case doctorPass:
		return "✅"
	case doctorWarn:
		return "⚠️ "
	case doctorFail:
		return "❌"
	default:
		return "⏭️ "
	}
}

func renderDoctorText(w io.Writer, report doctorReport) error {
	if _, err := fmt.Fprintf(w, "portr doctor (client %s)\nconfig: %s\n\n", report.ClientVersion, report.ConfigPath); err != nil {
		return err
	}

	counts := map[doctorStatus]int{}
	for _, check := range report.Checks {
		counts[check.Status]++

		if _, err := fmt.Fprintf(w, "%s %-20s %s\n", doctorStatusIcon(check.Status), check.Name, check.Detail); err != nil {
			return err
		}
		if check.Hint != "" {
			if _, err := fmt.Fprintf(w, "   → %s\n", check.Hint); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintf(w, "\n%d checks: %d passed, %d warnings, %d failed, %d skipped\n",
		len(report.Checks), counts[doctorPass], counts[doctorWarn], counts[doctorFail], counts[doctorSkipped])
	return err
}

func renderDoctorJSON(w io.Writer, report doctorReport) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
