package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/amalshaji/localport/internal/utils"
	"gopkg.in/yaml.v3"
)

type Tunnel struct {
	Name      string `yaml:"name"`
	Subdomain string `yaml:"subdomain"`
	Port      int    `yaml:"port"`
	Host      string `yaml:"host"`
}

func (t *Tunnel) setDefaults() {
	if t.Subdomain == "" {
		t.Subdomain = utils.GenerateTunnelSubdomain()
	}

	if t.Host == "" {
		t.Host = "localhost"
	}
}

func (t *Tunnel) GetAddr() string {
	return t.Host + ":" + fmt.Sprint(t.Port)
}

type Config struct {
	ServerUrl    string   `yaml:"serverUrl"`
	SshUrl       string   `yaml:"sshUrl"`
	TunnelUrl    string   `yaml:"tunnelUrl"`
	SecretKey    string   `yaml:"secretKey"`
	Tunnels      []Tunnel `yaml:"tunnels"`
	UseLocalHost bool     `yaml:"useLocalhost"`
	Debug        bool     `yaml:"debug"`
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
		c.Tunnels[i].setDefaults()
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

func (c *ClientConfig) GetAddr() string {
	protocol := "http"
	if !c.UseLocalHost {
		protocol = "https"
	}

	return protocol + "://" + c.Tunnel.Subdomain + "." + c.TunnelUrl
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
var DefaultConfigDir = homedir + "/.localport"
var DefaultConfigPath = DefaultConfigDir + "/config.yaml"
var DefaultKeyDir = DefaultConfigDir + "/keys"
var DefaultKeyPath = DefaultKeyDir + "/id_rsa"

func checkDefaultConfigFileExists() bool {
	_, err := os.Stat(DefaultConfigPath)
	return !os.IsNotExist(err)
}

func initConfig() error {
	if checkDefaultConfigFileExists() {
		return nil
	}

	_, err := os.Stat(DefaultConfigDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(DefaultConfigDir, os.ModePerm)
		if err != nil {
			return err
		}
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

func (c Config) ValidateConfig() error {
	if !checkDefaultConfigFileExists() {
		err := initConfig()
		if err != nil {
			return err
		}
	}

	payloadMap := map[string]string{
		"key": c.SecretKey,
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		c.GetAdminAddress()+"/config/validate",
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config validation failed")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	_, err = os.Stat(DefaultKeyDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(DefaultKeyDir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	err = os.WriteFile(DefaultKeyPath, body, 0644)
	if err != nil {
		return fmt.Errorf("failed to setup credentials: %s", err)
	}

	return nil
}
