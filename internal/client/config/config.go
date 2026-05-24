package config

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/gommon/color"
	"gopkg.in/yaml.v3"
)

var UNABLE_TO_OPEN_EDITOR = color.Yellow("Unable to open editor. Please edit the config file manually at " + DefaultConfigPath)

type Tunnel struct {
	Name                 string                   `yaml:"name"`
	Subdomain            string                   `yaml:"subdomain"`
	Port                 int                      `yaml:"port"`
	Host                 string                   `yaml:"host"`
	Type                 constants.ConnectionType `yaml:"type"`
	ResponseFormat       string                   `yaml:"response_format"`
	ResponseTemplate     string                   `yaml:"response_tmpl"`
	ResponseTemplateFile string                   `yaml:"response_tmpl_file"`
	RemotePort           int
	PoolSize             int `yaml:"pool_size"`
}

func (t *Tunnel) SetDefaults() {
	if t.Type == "" {
		t.Type = constants.Http
	}

	if t.Type == constants.Http && t.Subdomain == "" {
		t.Subdomain = utils.GenerateTunnelSubdomain()
	}

	if t.Host == "" && t.Type != constants.Stub {
		t.Host = "localhost"
	}

	if t.Type == constants.Stub {
		t.PoolSize = 1
	} else if t.PoolSize <= 0 {
		t.PoolSize = 2
	}
}

func (t *Tunnel) Validate() error {
	if t.Type == constants.Stub && strings.TrimSpace(t.Subdomain) == "" {
		return fmt.Errorf("subdomain is required for stub tunnels")
	}

	if t.Type == constants.Http || t.Type == constants.Stub {
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

	return nil
}

func (t *Tunnel) GetLocalAddr() string {
	return t.Host + ":" + fmt.Sprint(t.Port)
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

type Config struct {
	ServerUrl                  string   `yaml:"server_url"`
	WsUrl                      string   `yaml:"ws_url"`
	TunnelUrl                  string   `yaml:"tunnel_url"`
	SecretKey                  string   `yaml:"secret_key"`
	Tunnels                    []Tunnel `yaml:"tunnels"`
	UseLocalHost               bool     `yaml:"use_localhost"`
	Debug                      bool     `yaml:"debug"`
	UseVite                    bool     `yaml:"use_vite"`
	DashboardPort              int      `yaml:"dashboard_port"`
	DisableDashboard           bool     `yaml:"disable_dashboard"`
	EnableRequestLogging       *bool    `yaml:"enable_request_logging"`
	ConnectionLogRetentionDays int      `yaml:"connection_log_retention_days"`
	HealthCheckInterval        int      `yaml:"health_check_interval"`
	HealthCheckMaxRetries      int      `yaml:"health_check_max_retries"`
	DisableTUI                 bool     `yaml:"disable_tui"`
	EnableHttpReverseProxy     bool     `yaml:"enable_http_reverse_proxy"`
	DisableUpdateCheck         bool     `yaml:"disable_update_check"`
}

func (c *Config) SetDefaults() {
	if c.ServerUrl == "" {
		c.ServerUrl = "localhost:8000"
	}

	if c.TunnelUrl == "" {
		c.TunnelUrl = c.ServerUrl
	}

	if c.WsUrl == "" {
		c.WsUrl = c.TunnelUrl
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

	for _, tunnel := range c.Tunnels {
		if err := tunnel.Validate(); err != nil {
			return err
		}
	}

	return nil
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
	ServerUrl              string
	WsUrl                  string
	TunnelUrl              string
	SecretKey              string
	ConnectionID           string
	Tunnel                 Tunnel
	UseLocalHost           bool
	Debug                  bool
	EnableRequestLogging   bool
	HealthCheckInterval    int
	HealthCheckMaxRetries  int
	DisableTUI             bool
	DisableTerminalLogs    bool
	EnableHttpReverseProxy bool
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
	if c.Tunnel.Type == constants.Http || c.Tunnel.Type == constants.Stub {
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
	}

	return config, nil
}

var homedir, _ = os.UserHomeDir()
var DefaultConfigDir = homedir + "/.portr"
var DefaultConfigPath = DefaultConfigDir + "/config.yaml"

func checkDefaultConfigFileExists() bool {
	_, err := os.Stat(DefaultConfigPath)
	return !os.IsNotExist(err)
}

func initConfig() error {
	if checkDefaultConfigFileExists() {
		return nil
	}

	f, err := os.Create(DefaultConfigPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func EditConfig() error {
	if !checkDefaultConfigFileExists() {
		err := initConfig()
		if err != nil {
			return err
		}
	}

	var editorCmd string

	switch os := runtime.GOOS; os {
	case "darwin":
		editorCmd = "open"
	case "linux":
		editorCmd = "xdg-open"
	case "windows":
		editorCmd = "start"
	default:
		fmt.Println(UNABLE_TO_OPEN_EDITOR)
		return nil
	}

	cmd := exec.Command(editorCmd, DefaultConfigPath)
	if err := cmd.Run(); err != nil {
		fmt.Println(UNABLE_TO_OPEN_EDITOR)
		return nil
	}

	return nil
}

func SetConfig(config string) error {
	if !checkDefaultConfigFileExists() {
		err := initConfig()
		if err != nil {
			return err
		}
	}

	return os.WriteFile(DefaultConfigPath, []byte(config), 0644)
}

func GetConfig(token string, remote string) error {
	payloadMap := map[string]string{
		"secret_key": token,
	}

	client := resty.New()

	if !(strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "https://")) {
		if strings.HasPrefix(remote, "localhost:") {
			remote = "http://" + remote
		} else {
			remote = "https://" + remote
		}
	}

	var response struct {
		Message string `json:"message"`
	}

	resp, err := client.R().SetError(&response).SetResult(&response).SetBody(payloadMap).Post(remote + "/api/v1/config/download")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("%s", response.Message)
	}

	return SetConfig(response.Message)
}
