package clientconfig

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"gopkg.in/yaml.v3"
)

type Tunnel struct {
	Name                 string                   `yaml:"name"`
	Subdomain            string                   `yaml:"subdomain"`
	Port                 int                      `yaml:"port"`
	Host                 string                   `yaml:"host"`
	Dir                  string                   `yaml:"dir"`
	Type                 constants.ConnectionType `yaml:"type"`
	ResponseFormat       string                   `yaml:"response_format"`
	ResponseTemplate     string                   `yaml:"response_tmpl"`
	ResponseTemplateFile string                   `yaml:"response_tmpl_file"`
	RemotePort           int
	PoolSize             int    `yaml:"pool_size"`
	HostHeader           string `yaml:"host_header"`
	BasicAuth            string `yaml:"basic_auth"`
}

// HostHeaderRewrite sets the outbound Host header to the local address.
const HostHeaderRewrite = "rewrite"

func (t *Tunnel) SetDefaults() {
	if t.Type == "" {
		t.Type = constants.Http
	}

	if (t.Type == constants.Http || t.Type == constants.Static) && t.Subdomain == "" {
		t.Subdomain = utils.GenerateTunnelSubdomain()
	}
	if t.Type.IsHTTPLike() {
		t.Subdomain = utils.NormalizeSubdomain(t.Subdomain)
	}

	// Stub and static tunnels are served by an in-process responder, so their
	// local address is assigned when that responder starts.
	if t.Host == "" && t.Type != constants.Stub && t.Type != constants.Static {
		t.Host = "localhost"
	}

	if t.Type == constants.Stub || t.Type == constants.Static {
		t.PoolSize = 1
	} else if t.PoolSize <= 0 {
		t.PoolSize = 2
	}
}

func (t *Tunnel) Validate() error {
	if strings.TrimSpace(t.HostHeader) != "" && t.Type != constants.Http {
		return fmt.Errorf("host_header is only supported for http tunnels")
	}

	if err := t.validateBasicAuth(); err != nil {
		return err
	}

	if t.Type == constants.Stub && strings.TrimSpace(t.Subdomain) == "" {
		return fmt.Errorf("subdomain is required for stub tunnels")
	}

	if t.Type.IsHTTPLike() {
		if err := utils.ValidateSubdomain(t.Subdomain); err != nil {
			return err
		}
	}

	if t.Type == constants.Stub {
		if strings.TrimSpace(t.ResponseFormat) == "" {
			return fmt.Errorf("response_format is required for stub tunnels")
		}
		if strings.TrimSpace(t.ResponseTemplate) == "" {
			return fmt.Errorf("response_tmpl or response_tmpl_file is required for stub tunnels")
		}
	}

	if t.Type == constants.Static {
		if strings.TrimSpace(t.Dir) == "" {
			return fmt.Errorf("dir is required for static tunnels")
		}
		info, err := os.Stat(t.Dir)
		if err != nil {
			return fmt.Errorf("failed to read dir %q: %w", t.Dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("dir %q is not a directory", t.Dir)
		}
	}

	return nil
}

func (t *Tunnel) GetLocalAddr() string {
	return net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
}

// ResolvedHostHeader returns the Host header to send to the local service, or
// an empty string when the public Host should be passed through unchanged.
func (t *Tunnel) ResolvedHostHeader(localAddr string) string {
	value := strings.TrimSpace(t.HostHeader)
	switch {
	case value == "":
		return ""
	case strings.EqualFold(value, HostHeaderRewrite):
		return localAddr
	default:
		return value
	}
}

// ResolvedBasicAuth returns the user:password credential required to reach the
// tunnel, or an empty string when the tunnel is unprotected. The credential
// stays opaque here: only validateBasicAuth needs to look inside it, and the
// gate compares it whole. Only the outer value is trimmed, so whitespace inside
// either half survives ("user: pass" is a password with a leading space).
func (t *Tunnel) ResolvedBasicAuth() string {
	return strings.TrimSpace(t.BasicAuth)
}

// validateBasicAuth rejects credentials that look like protection but are not.
// The split is on the first colon only, matching RFC 7617: a userid cannot
// contain a colon, a password can.
func (t *Tunnel) validateBasicAuth() error {
	value := t.ResolvedBasicAuth()
	if value == "" {
		return nil
	}

	if !t.Type.IsHTTPLike() {
		return fmt.Errorf("basic_auth is not supported for %s tunnels", t.Type)
	}

	username, password, found := strings.Cut(value, ":")
	switch {
	case !found:
		return fmt.Errorf("basic_auth must be in user:password format")
	case username == "":
		return fmt.Errorf("basic_auth username cannot be empty")
	case password == "":
		return fmt.Errorf("basic_auth password cannot be empty")
	}

	return nil
}

// ResolveServeDir makes a static tunnel's directory absolute, relative to
// baseDir. It deliberately does not touch the filesystem: Load runs on every
// CLI invocation, so a stale static entry must not break unrelated commands.
// Validate is where the directory is checked to exist.
func (t *Tunnel) ResolveServeDir(baseDir string) error {
	if t.Type != constants.Static {
		return nil
	}

	dir := strings.TrimSpace(t.Dir)
	if dir == "" {
		return fmt.Errorf("dir is required for static tunnels")
	}
	if !filepath.IsAbs(dir) {
		if baseDir == "" {
			baseDir = "."
		}
		dir = filepath.Join(baseDir, dir)
	}

	resolved, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve dir %q: %w", t.Dir, err)
	}
	t.Dir = resolved
	return nil
}

// StatusKey identifies a tunnel in TUI messages. Stub and static tunnels share
// one local responder port, so they key on subdomain instead.
func (t Tunnel) StatusKey() string {
	if t.Type == constants.Stub || t.Type == constants.Static {
		return string(t.Type) + ":" + t.Subdomain
	}
	return fmt.Sprint(t.Port)
}

// DisplayName is the human-readable label for a tunnel.
func (t Tunnel) DisplayName() string {
	if t.Name != "" {
		return t.Name
	}
	if (t.Type == constants.Stub || t.Type == constants.Static) && t.Subdomain != "" {
		return t.Subdomain
	}
	return fmt.Sprint(t.Port)
}

func (t *Tunnel) ValidateStubTemplateSource() error {
	if t.Type != constants.Stub {
		return nil
	}

	hasInline := strings.TrimSpace(t.ResponseTemplate) != ""
	hasFile := strings.TrimSpace(t.ResponseTemplateFile) != ""

	switch {
	case hasInline && hasFile:
		return fmt.Errorf("only one of response_tmpl or response_tmpl_file can be provided for stub tunnels")
	case !hasInline && !hasFile:
		return fmt.Errorf("response_tmpl or response_tmpl_file is required for stub tunnels")
	default:
		return nil
	}
}

func (t *Tunnel) ResolveStubTemplate(baseDir string) error {
	if t.Type != constants.Stub {
		return nil
	}

	if err := t.ValidateStubTemplateSource(); err != nil {
		return err
	}

	if strings.TrimSpace(t.ResponseTemplateFile) == "" {
		return nil
	}

	templatePath := t.ResponseTemplateFile
	if !filepath.IsAbs(templatePath) {
		if baseDir == "" {
			baseDir = "."
		}
		templatePath = filepath.Join(baseDir, templatePath)
	}

	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read response_tmpl_file %q: %w", t.ResponseTemplateFile, err)
	}

	t.ResponseTemplate = string(templateBytes)
	if strings.TrimSpace(t.ResponseTemplate) == "" {
		return fmt.Errorf("response_tmpl_file %q is empty", t.ResponseTemplateFile)
	}

	return nil
}

var DefaultRedactHeaders = []string{
	"Authorization",
	"Cookie",
	"Set-Cookie",
	"Proxy-Authorization",
}

type Config struct {
	ServerUrl                       string              `yaml:"server_url"`
	SshUrl                          string              `yaml:"ssh_url"`
	TunnelUrl                       string              `yaml:"tunnel_url"`
	SecretKey                       string              `yaml:"secret_key"`
	Tunnels                         []Tunnel            `yaml:"tunnels"`
	Groups                          map[string][]string `yaml:"groups"`
	UseLocalHost                    bool                `yaml:"use_localhost"`
	Debug                           bool                `yaml:"debug"`
	UseVite                         bool                `yaml:"use_vite"`
	DashboardPort                   int                 `yaml:"dashboard_port"`
	DisableDashboard                bool                `yaml:"disable_dashboard"`
	EnableRequestLogging            *bool               `yaml:"enable_request_logging"`
	RedactHeaders                   []string            `yaml:"redact_headers"`
	ConnectionLogRetentionDays      int                 `yaml:"connection_log_retention_days"`
	HealthCheckInterval             int                 `yaml:"health_check_interval"`
	HealthCheckMaxRetries           int                 `yaml:"health_check_max_retries"`
	DisableTUI                      bool                `yaml:"disable_tui"`
	EnableQRCode                    *bool               `yaml:"enable_qr_code"`
	DisableUpdateCheck              bool                `yaml:"disable_update_check"`
	InsecureSkipHostKeyVerification *bool               `yaml:"insecure_skip_host_key_verification"`
}

func (c *Config) SetDefaults() {
	if c.ServerUrl == "" {
		c.ServerUrl = "localhost:8000"
	}

	if c.SshUrl == "" {
		c.SshUrl = c.ServerUrl
	}

	if c.TunnelUrl == "" {
		c.TunnelUrl = c.ServerUrl
	}

	if c.DashboardPort == 0 {
		c.DashboardPort = DefaultDashboardPort
	}

	if c.HealthCheckInterval == 0 {
		c.HealthCheckInterval = 3
	}

	if c.HealthCheckMaxRetries == 0 {
		c.HealthCheckMaxRetries = 10
	}

	if c.EnableRequestLogging == nil {
		defaultValue := true
		c.EnableRequestLogging = &defaultValue
	}

	if c.EnableQRCode == nil {
		defaultValue := true
		c.EnableQRCode = &defaultValue
	}

	if len(c.RedactHeaders) == 0 {
		c.RedactHeaders = append([]string(nil), DefaultRedactHeaders...)
	}

	if c.InsecureSkipHostKeyVerification == nil {
		defaultValue := true
		c.InsecureSkipHostKeyVerification = &defaultValue
	}

	for i := range c.Tunnels {
		c.Tunnels[i].SetDefaults()
	}
}

func (c Config) Validate() error {
	if c.ConnectionLogRetentionDays < 0 {
		return fmt.Errorf("connection_log_retention_days must be greater than or equal to 0")
	}

	if !c.DisableDashboard && (c.DashboardPort < 1 || c.DashboardPort > 65535) {
		return fmt.Errorf("dashboard_port must be between 1 and 65535")
	}

	tunnelNames := make(map[string]bool, len(c.Tunnels))
	for _, tunnel := range c.Tunnels {
		if err := tunnel.Validate(); err != nil {
			return err
		}
		if tunnel.Name != "" {
			tunnelNames[tunnel.Name] = true
		}
	}

	for group, members := range c.Groups {
		if tunnelNames[group] {
			return fmt.Errorf("group %q conflicts with a tunnel of the same name", group)
		}
		if len(members) == 0 {
			return fmt.Errorf("group %q must reference at least one tunnel", group)
		}
		for _, member := range members {
			if !tunnelNames[member] {
				return fmt.Errorf("group %q references unknown tunnel %q", group, member)
			}
		}
	}

	return nil
}

// templateKeys are the only top-level keys a team template may set. Everything
// else is a connection setting owned by the server or a machine-local
// preference, and would be overwritten on the next `portr config pull`.
var templateKeys = []string{"tunnels", "groups"}

// documentMapping returns the top-level mapping of a single YAML document, or
// nil when the input is blank or only comments.
func documentMapping(raw string) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(strings.NewReader(raw))

	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}

	if err := decoder.Decode(new(yaml.Node)); err == nil {
		return nil, fmt.Errorf("expected a single YAML document")
	} else if !errors.Is(err, io.EOF) {
		return nil, err
	}

	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a YAML mapping")
	}

	return document.Content[0], nil
}

// ValidateTemplate checks a team-managed config fragment. An empty template is
// valid and means the team has not configured one.
func ValidateTemplate(raw string) error {
	mapping, err := documentMapping(raw)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}
	if mapping == nil {
		return nil
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		key := mapping.Content[i].Value
		if !slices.Contains(templateKeys, key) {
			return fmt.Errorf("%q is not allowed in a team template; only %s can be set",
				key, strings.Join(templateKeys, " and "))
		}
	}

	var config Config
	if err := mapping.Decode(&config); err != nil {
		return err
	}

	names := make(map[string]bool, len(config.Tunnels))
	for _, tunnel := range config.Tunnels {
		if strings.TrimSpace(tunnel.Name) == "" {
			return fmt.Errorf("every tunnel in a team template needs a name")
		}
		if names[tunnel.Name] {
			return fmt.Errorf("duplicate tunnel name %q", tunnel.Name)
		}
		names[tunnel.Name] = true

		if err := validateTemplateTunnel(tunnel); err != nil {
			return err
		}
	}

	config.SetDefaults()
	return config.Validate()
}

// validateTemplateTunnel rejects tunnel settings that are wrong when shared
// verbatim: a path on the author's machine, which means nothing on a
// teammate's machine and which the server validating the template has no
// business resolving, or a credential, which a shared plaintext template
// cannot keep secret.
func validateTemplateTunnel(tunnel Tunnel) error {
	switch {
	case tunnel.Type == constants.Static:
		return fmt.Errorf("tunnel %q: static tunnels serve a directory on your machine and cannot be shared in a team template", tunnel.Name)
	case strings.TrimSpace(tunnel.ResponseTemplateFile) != "":
		return fmt.Errorf("tunnel %q: response_tmpl_file points at a path on your machine; inline the body with response_tmpl instead", tunnel.Name)
	case strings.TrimSpace(tunnel.BasicAuth) != "":
		return fmt.Errorf("tunnel %q: basic_auth is a credential and cannot be shared in a team template; set it in your local config or pass --basic-auth", tunnel.Name)
	}

	return nil
}

// ReplaceTunnels swaps in tunnels defined outside the config file. Groups are
// dropped because they can only reference tunnels from the config file.
func (c *Config) ReplaceTunnels(tunnels ...Tunnel) {
	c.Tunnels = tunnels
	c.Groups = nil
}

// SelectTunnels returns the tunnels matching the given tunnel or group names.
// Passing no names selects every tunnel.
func (c Config) SelectTunnels(services []string) []Tunnel {
	if len(services) == 0 {
		return c.Tunnels
	}

	names := c.expandGroups(services)

	var selected []Tunnel
	for _, tunnel := range c.Tunnels {
		if slices.Contains(names, tunnel.Name) {
			selected = append(selected, tunnel)
		}
	}
	return selected
}

func (c Config) expandGroups(services []string) []string {
	expanded := make([]string, 0, len(services))
	for _, service := range services {
		if members, ok := c.Groups[service]; ok {
			expanded = append(expanded, members...)
			continue
		}
		expanded = append(expanded, service)
	}
	return expanded
}

func (c Config) NoMatchingTunnelsError(services []string) error {
	var available []string
	for _, tunnel := range c.Tunnels {
		if tunnel.Name != "" {
			available = append(available, tunnel.Name)
		}
	}
	for group := range c.Groups {
		available = append(available, group+" (group)")
	}

	if len(available) == 0 {
		return fmt.Errorf("no named tunnels configured; run `portr config edit` to add one")
	}
	slices.Sort(available)

	return fmt.Errorf("no tunnel or group matched %s; available: %s",
		strings.Join(services, ", "), strings.Join(available, ", "))
}

func (c Config) GetAdminAddress() string {
	protocol := "http"
	if !c.UseLocalHost {
		protocol = "https"
	}

	return protocol + "://" + c.ServerUrl
}

func (c Config) GetDashboardAddress() string {
	if c.DisableDashboard {
		return ""
	}

	return fmt.Sprintf("http://localhost:%d", c.DashboardPort)
}

func (c Config) GetDashboardDisableLabel() string {
	if !c.DisableDashboard {
		return ""
	}

	return "disabled via config"
}

type ClientConfig struct {
	ServerUrl                       string
	SshUrl                          string
	TunnelUrl                       string
	SecretKey                       string
	ConnectionID                    string
	Tunnel                          Tunnel
	UseLocalHost                    bool
	Debug                           bool
	EnableRequestLogging            bool
	RedactHeaders                   []string
	HealthCheckInterval             int
	HealthCheckMaxRetries           int
	DisableTUI                      bool
	DisableTerminalLogs             bool
	InsecureSkipHostKeyVerification bool
}

// ClientConfigForTunnel builds the per-tunnel config handed to the tunnel
// client. It expects a loaded Config, i.e. one that went through SetDefaults.
// The redaction list is resolved here so every consumer inherits the same
// guarantee: a tunnel that enforces basic auth always redacts Authorization,
// even when a custom redact_headers list omits it, so captures can never
// persist the tunnel's own credential.
func (c *Config) ClientConfigForTunnel(tunnel Tunnel) ClientConfig {
	return ClientConfig{
		ServerUrl:                       c.ServerUrl,
		SshUrl:                          c.SshUrl,
		TunnelUrl:                       c.TunnelUrl,
		SecretKey:                       c.SecretKey,
		Tunnel:                          tunnel,
		UseLocalHost:                    c.UseLocalHost,
		Debug:                           c.Debug,
		EnableRequestLogging:            *c.EnableRequestLogging,
		RedactHeaders:                   c.redactHeadersForTunnel(tunnel),
		HealthCheckInterval:             c.HealthCheckInterval,
		HealthCheckMaxRetries:           c.HealthCheckMaxRetries,
		DisableTUI:                      c.DisableTUI,
		InsecureSkipHostKeyVerification: *c.InsecureSkipHostKeyVerification,
	}
}

func (c *Config) redactHeadersForTunnel(tunnel Tunnel) []string {
	names := append([]string(nil), c.RedactHeaders...)
	if tunnel.ResolvedBasicAuth() == "" {
		return names
	}

	hasAuthorization := slices.ContainsFunc(names, func(name string) bool {
		return strings.EqualFold(name, "Authorization")
	})
	if hasAuthorization {
		return names
	}

	return append(names, "Authorization")
}

const DefaultDashboardPort = 7777

func (c *ClientConfig) GetHttpTunnelAddr() string {
	protocol := "http"
	if !c.UseLocalHost {
		protocol = "https"
	}

	return protocol + "://" + c.Tunnel.Subdomain + "." + c.TunnelUrl
}

func (c *ClientConfig) GetStubTunnelAddr() string {
	return c.GetHttpTunnelAddr()
}

func (c *ClientConfig) GetTcpTunnelAddr() string {
	split := strings.Split(c.TunnelUrl, ":")
	return split[0] + ":" + fmt.Sprint(c.Tunnel.RemotePort)
}

func (c *ClientConfig) GetTunnelAddr() string {
	if c.Tunnel.Type.IsHTTPLike() {
		return c.GetHttpTunnelAddr()
	}
	return c.GetTcpTunnelAddr()
}

func (c *ClientConfig) GetServerAddr() string {
	protocol := "http"
	if !c.UseLocalHost {
		protocol = "https"
	}

	return protocol + "://" + c.ServerUrl
}

func Load(configFile string) (Config, error) {
	var config Config

	bytes, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, err
	}

	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return Config{}, err
	}

	config.SetDefaults()

	baseDir := filepath.Dir(configFile)
	for i := range config.Tunnels {
		if err := config.Tunnels[i].ResolveStubTemplate(baseDir); err != nil {
			return Config{}, err
		}
		if err := config.Tunnels[i].ResolveServeDir(baseDir); err != nil {
			return Config{}, err
		}
	}

	return config, nil
}
