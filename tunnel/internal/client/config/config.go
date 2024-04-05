package config

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v3"
)

type Tunnel struct {
	Name      string                   `yaml:"name"`
	Subdomain string                   `yaml:"subdomain"`
	Port      int                      `yaml:"port"`
	Host      string                   `yaml:"host"`
	Type      constants.ConnectionType `yaml:"type"`
}

func (t *Tunnel) SetDefaults() {
	if t.Type == "" {
		t.Type = constants.Http
	}

	if t.Type == constants.Http && t.Subdomain == "" {
		t.Subdomain = utils.GenerateTunnelSubdomain()
	}

	if t.Host == "" {
		t.Host = "localhost"
	}
}

func (t *Tunnel) GetLocalAddr() string {
	return t.Host + ":" + fmt.Sprint(t.Port)
}

type Config struct {
	ServerUrl    string   `yaml:"server_url"`
	SshUrl       string   `yaml:"ssh_url"`
	TunnelUrl    string   `yaml:"tunnel_url"`
	SecretKey    string   `yaml:"secret_key"`
	Tunnels      []Tunnel `yaml:"tunnels"`
	UseLocalHost bool     `yaml:"use_localhost"`
	Debug        bool     `yaml:"debug"`
	UseVite      bool     `yaml:"use_vite"`
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

	for i := range c.Tunnels {
		c.Tunnels[i].SetDefaults()
	}
}

func (c Config) GetAdminAddress() string {
	protocol := "http"
	if !c.UseLocalHost {
		protocol = "https"
	}

	return protocol + "://" + c.ServerUrl
}

type ClientConfig struct {
	ServerUrl    string
	SshUrl       string
	TunnelUrl    string
	SecretKey    string
	Tunnel       Tunnel
	UseLocalHost bool
	Debug        bool
}

func (c *ClientConfig) GetHttpTunnelAddr() string {
	protocol := "http"
	if !c.UseLocalHost {
		protocol = "https"
	}

	return protocol + "://" + c.Tunnel.Subdomain + "." + c.TunnelUrl
}

func (c *ClientConfig) GetTcpTunnelAddr(port int) string {
	split := strings.Split(c.TunnelUrl, ":")
	return split[0] + ":" + fmt.Sprint(port)
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
		return fmt.Errorf("unsupported platform")
	}

	cmd := exec.Command(editorCmd, DefaultConfigPath)
	if err := cmd.Run(); err != nil {
		return err
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
		return fmt.Errorf(response.Message)
	}

	return SetConfig(response.Message)
}
