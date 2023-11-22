package config

import (
	"fmt"
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
	ServerUrl string   `yaml:"serverUrl"`
	SshUrl    string   `yaml:"sshUrl"`
	TunnelUrl string   `yaml:"tunnelUrl"`
	Secretkey string   `yaml:"secretkey"`
	Tunnels   []Tunnel `yaml:"tunnels"`
	Secure    bool     `yaml:"secure"`
	Debug     bool     `yaml:"debug"`
}

func (c *Config) SetDefaults() {
	if c.ServerUrl == "" {
		c.ServerUrl = "localhost:8000"
	}

	if c.SshUrl == "" {
		c.SshUrl = "localhost:2222"
	}

	if c.TunnelUrl == "" {
		c.TunnelUrl = "localhost:8001"
	}

	for i := range c.Tunnels {
		c.Tunnels[i].setDefaults()
	}
}

type ClientConfig struct {
	ServerUrl string
	SshUrl    string
	TunnelUrl string
	Secretkey string
	Tunnel    Tunnel
	Secure    bool
	Debug     bool
}

func (c *ClientConfig) GetAddr() string {
	protocol := "http"
	if c.Secure {
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
