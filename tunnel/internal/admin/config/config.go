package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

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

func Load() *AdminConfig {
	portStr := os.Getenv("PORTR_ADMIN_PORT")
	if portStr == "" {
		portStr = "8000"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("Invalid PORTR_ADMIN_PORT:", err)
	}

	domain := os.Getenv("PORTR_DOMAIN")
	if domain == "" {
		domain = "localhost:8000"
	}

	serverURL := os.Getenv("PORTR_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8001"
	}

	sshURL := os.Getenv("PORTR_SSH_URL")
	if sshURL == "" {
		sshURL = "localhost:2222"
	}

	fmt.Println("PORTR_ADMIN_USE_VITE:", os.Getenv("PORTR_ADMIN_USE_VITE"))

	return &AdminConfig{
		Port:           port,
		Domain:         domain,
		Debug:          os.Getenv("PORTR_ADMIN_DEBUG") == "true",
		UseVite:        os.Getenv("PORTR_ADMIN_USE_VITE") == "true",
		GithubClientID: os.Getenv("PORTR_ADMIN_GITHUB_CLIENT_ID"),
		GithubSecret:   os.Getenv("PORTR_ADMIN_GITHUB_CLIENT_SECRET"),
		ServerURL:      serverURL,
		SshURL:         sshURL,
	}
}

func (c *AdminConfig) DomainAddress() string {
	if c.Debug || c.Domain == "localhost:8000" {
		return "http://" + c.Domain
	}
	return "https://" + c.Domain
}
