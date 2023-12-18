package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type OAuth struct {
	ClientID     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
}

type AdminConfig struct {
	Host    string
	Port    int
	UseVite bool  `yaml:"useVite"`
	OAuth   OAuth `yaml:"oauth"`
}

func (a AdminConfig) Address() string {
	return a.Host + ":" + fmt.Sprint(a.Port)
}

func (a AdminConfig) ListenAddress() string {
	return ":" + fmt.Sprint(a.Port)
}

type SshConfig struct {
	Host    string
	Port    int
	KeysDir string
}

func (s SshConfig) Address() string {
	return s.Host + ":" + fmt.Sprint(s.Port)
}

type ProxyConfig struct {
	Host string
	Port int
}

func (p ProxyConfig) Address() string {
	return p.Host + ":" + fmt.Sprint(p.Port)
}

type Config struct {
	Admin        AdminConfig `yaml:"admin"`
	Ssh          SshConfig   `yaml:"ssh"`
	Proxy        ProxyConfig `yaml:"proxy"`
	Domain       string      `yaml:"domain"`
	UseLocalHost bool        `yaml:"useLocalhost"`
	Debug        bool        `yaml:"debug"`
}

func new() *Config {
	return &Config{
		Admin: AdminConfig{
			Host:    "localhost",
			Port:    8000,
			UseVite: false,
			OAuth: OAuth{
				ClientID:     "",
				ClientSecret: "",
			},
		},
		Ssh: SshConfig{
			Host:    "localhost",
			Port:    2222,
			KeysDir: "./keys",
		},
		Proxy: ProxyConfig{
			Host: "localhost",
			Port: 8001,
		},
		Domain:       "",
		UseLocalHost: false,
		Debug:        false,
	}
}

func (c *Config) setDefaults() {
	if c.UseLocalHost {
		c.Domain = c.Admin.Address()
	}
}

func (c Config) Protocol() string {
	if !c.UseLocalHost {
		return "https"
	}
	return "http"
}

func (c Config) AdminUrl() string {
	if !c.UseLocalHost {
		return "https://" + c.Domain
	}
	return "http://" + c.Admin.Address()
}

func (c Config) ExtractSubdomain(url string) string {
	withoutProtocol := strings.ReplaceAll(url, c.Protocol()+"://", "")
	if !c.UseLocalHost {
		return strings.ReplaceAll(withoutProtocol, "."+c.Domain, "")
	}
	return strings.ReplaceAll(withoutProtocol, "."+c.Proxy.Address(), "")
}

func Load(path string) (*Config, error) {
	c := new()

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal([]byte(os.ExpandEnv(string(bytes))), c)
	if err != nil {
		return nil, err
	}

	c.setDefaults()

	return c, nil
}
