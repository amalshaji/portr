package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

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

type DatabaseConfig struct {
	Url         string
	Driver      string
	AutoMigrate bool
}

type Config struct {
	Ssh          SshConfig
	Proxy        ProxyConfig
	Domain       string
	UseLocalHost bool
	Debug        bool
	Database     DatabaseConfig
}

func new() *Config {
	sshPortStr := os.Getenv("SSH_PORT")
	if sshPortStr == "" {
		sshPortStr = "2222"
	}
	sshPort, err := strconv.Atoi(sshPortStr)
	if err != nil {
		log.Fatal(err)
	}

	proxyPortStr := os.Getenv("PROXY_PORT")
	if proxyPortStr == "" {
		proxyPortStr = "8001"
	}
	proxyPort, err := strconv.Atoi(proxyPortStr)
	if err != nil {
		log.Fatal(err)
	}

	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "localhost:8000"
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatal("DB_URL is required")
	}

	dbDriver := strings.Split(os.Getenv("DB_URL"), "://")[0]

	return &Config{
		Ssh: SshConfig{
			Host: "localhost",
			Port: sshPort,
		},
		Proxy: ProxyConfig{
			Host: "localhost",
			Port: proxyPort,
		},
		Domain:       domain,
		UseLocalHost: os.Getenv("USE_LOCALHOST") == "true",
		Debug:        os.Getenv("DEBUG") == "true",
		Database: DatabaseConfig{
			Url:    dbUrl,
			Driver: dbDriver,
		},
	}
}

func (c *Config) HttpTunnelUrl(subdomain string) string {
	if !c.UseLocalHost {
		return "https://" + subdomain + "." + c.Domain
	}
	return "http://" + subdomain + "." + c.Proxy.Address()
}

func (c *Config) TcpTunnelUrl(port uint32) string {
	if !c.UseLocalHost {
		return c.Domain + ":" + fmt.Sprint(port)
	}
	return "localhost:" + fmt.Sprint(port)
}

func (c Config) Protocol() string {
	if !c.UseLocalHost {
		return "https"
	}
	return "http"
}

func (c Config) ExtractSubdomain(url string) string {
	withoutProtocol := strings.ReplaceAll(url, c.Protocol()+"://", "")
	if !c.UseLocalHost {
		return strings.ReplaceAll(withoutProtocol, "."+c.Domain, "")
	}
	return strings.ReplaceAll(withoutProtocol, "."+c.Proxy.Address(), "")
}

func Load(path string) *Config {
	return new()
}
