package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type AdminConfig struct {
	Host string
	Port int
}

func (a AdminConfig) Address() string {
	return a.Host + ":" + fmt.Sprint(a.Port)
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
	Admin  AdminConfig `yaml:"admin"`
	Ssh    SshConfig   `yaml:"ssh"`
	Proxy  ProxyConfig `yaml:"proxy"`
	Domain string      `yaml:"domain"`
	Secure bool        `yaml:"secure"`
	Debug  bool        `yaml:"debug"`
}

func new() *Config {
	return &Config{
		Admin: AdminConfig{
			Host: "localhost",
			Port: 8000,
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
		Domain: "",
		Secure: false,
		Debug:  false,
	}
}

func (c Config) ExtractSubdomain(url string) string {
	// can be better
	protocol := "http"
	if c.Secure {
		protocol = "https"
	}

	withoutProtocol := strings.ReplaceAll(url, protocol+"://", "")

	if c.Secure {
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

	err = yaml.Unmarshal(bytes, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
