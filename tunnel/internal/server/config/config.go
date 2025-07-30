package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	_ "github.com/joho/godotenv/autoload"
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

type AdminConfig struct {
	Port           int
	Domain         string
	Debug          bool
	UseVite        bool
	GithubClientID string
	GithubSecret   string
	ServerURL      string
	SshURL         string
}

func (c *AdminConfig) DomainAddress() string {
	if c.Debug || c.Domain == "localhost:8000" {
		return "http://" + c.Domain
	}
	return "https://" + c.Domain
}

type Config struct {
	Ssh          SshConfig
	Proxy        ProxyConfig
	Domain       string
	UseLocalHost bool
	Debug        bool
	Database     DatabaseConfig
	Admin        AdminConfig
}

func new() *Config {
	sshPortStr := os.Getenv("PORTR_SSH_PORT")
	if sshPortStr == "" {
		sshPortStr = "2222"
	}
	sshPort, err := strconv.Atoi(sshPortStr)
	if err != nil {
		log.Fatal("Invalid PORTR_SSH_PORT", "port", sshPortStr, "error", err)
	}

	proxyPortStr := os.Getenv("PORTR_PROXY_PORT")
	if proxyPortStr == "" {
		proxyPortStr = "8001"
	}
	proxyPort, err := strconv.Atoi(proxyPortStr)
	if err != nil {
		log.Fatal("Invalid PORTR_PROXY_PORT", "port", proxyPortStr, "error", err)
	}

	domain := os.Getenv("PORTR_DOMAIN")
	if domain == "" {
		domain = "localhost:8001"
	}

	dbUrl := os.Getenv("PORTR_DB_URL")
	if dbUrl == "" {
		log.Fatal("PORTR_DB_URL is required")
	}

	dbDriver := strings.Split(os.Getenv("PORTR_DB_URL"), "://")[0]

	// Admin configuration
	adminPortStr := os.Getenv("PORTR_ADMIN_PORT")
	if adminPortStr == "" {
		adminPortStr = "8000"
	}
	adminPort, err := strconv.Atoi(adminPortStr)
	if err != nil {
		log.Fatal("Invalid PORTR_ADMIN_PORT", "port", adminPortStr, "error", err)
	}

	adminDomain := os.Getenv("PORTR_DOMAIN")
	if adminDomain == "" {
		adminDomain = "localhost:8000"
	}

	serverURL := os.Getenv("PORTR_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8001"
	}

	sshURL := os.Getenv("PORTR_SSH_URL")
	if sshURL == "" {
		sshURL = "localhost:2222"
	}

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
		UseLocalHost: os.Getenv("PORTR_TUNNEL_USE_LOCALHOST") == "true",
		Debug:        os.Getenv("PORTR_TUNNEL_DEBUG") == "true",
		Database: DatabaseConfig{
			Url:         dbUrl,
			Driver:      dbDriver,
			AutoMigrate: os.Getenv("PORTR_AUTO_MIGRATE") == "true",
		},
		Admin: AdminConfig{
			Port:           adminPort,
			Domain:         adminDomain,
			Debug:          os.Getenv("PORTR_ADMIN_DEBUG") == "true",
			UseVite:        os.Getenv("PORTR_ADMIN_USE_VITE") == "true",
			GithubClientID: os.Getenv("PORTR_ADMIN_GITHUB_CLIENT_ID"),
			GithubSecret:   os.Getenv("PORTR_ADMIN_GITHUB_CLIENT_SECRET"),
			ServerURL:      serverURL,
			SshURL:         sshURL,
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
