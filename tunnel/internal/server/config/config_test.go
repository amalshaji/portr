package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSshConfig_Address(t *testing.T) {
	config := SshConfig{
		Host: "localhost",
		Port: 2222,
	}

	assert.Equal(t, "localhost:2222", config.Address())
}

func TestProxyConfig_Address(t *testing.T) {
	config := ProxyConfig{
		Host: "localhost",
		Port: 8001,
	}

	assert.Equal(t, "localhost:8001", config.Address())
}

func TestConfig_HttpTunnelUrl(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		subdomain    string
		expectedUrl  string
	}{
		{
			name: "localhost setup",
			config: Config{
				Domain: "localhost:8001",
				Proxy: ProxyConfig{Host: "localhost", Port: 8001},
				UseLocalHost: true,
			},
			subdomain:   "test",
			expectedUrl: "http://test.localhost:8001",
		},
		{
			name: "production setup",
			config: Config{
				Domain: "example.com",
				UseLocalHost: false,
			},
			subdomain:   "test",
			expectedUrl: "https://test.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HttpTunnelUrl(tt.subdomain)
			assert.Equal(t, tt.expectedUrl, result)
		})
	}
}

func TestConfig_TcpTunnelUrl(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		port         uint32
		expectedUrl  string
	}{
		{
			name: "localhost setup",
			config: Config{
				UseLocalHost: true,
			},
			port:        9000,
			expectedUrl: "localhost:9000",
		},
		{
			name: "production setup",
			config: Config{
				Domain:       "example.com",
				UseLocalHost: false,
			},
			port:        9000,
			expectedUrl: "example.com:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.TcpTunnelUrl(tt.port)
			assert.Equal(t, tt.expectedUrl, result)
		})
	}
}

func TestConfig_Protocol(t *testing.T) {
	tests := []struct {
		name             string
		useLocalHost     bool
		expectedProtocol string
	}{
		{
			name:             "localhost",
			useLocalHost:     true,
			expectedProtocol: "http",
		},
		{
			name:             "production",
			useLocalHost:     false,
			expectedProtocol: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{UseLocalHost: tt.useLocalHost}
			assert.Equal(t, tt.expectedProtocol, config.Protocol())
		})
	}
}

func TestConfig_ExtractSubdomain(t *testing.T) {
	tests := []struct {
		name              string
		config            Config
		url               string
		expectedSubdomain string
	}{
		{
			name: "localhost",
			config: Config{
				UseLocalHost: true,
				Proxy:        ProxyConfig{Host: "localhost", Port: 8001},
			},
			url:               "http://test.localhost:8001",
			expectedSubdomain: "test",
		},
		{
			name: "production",
			config: Config{
				Domain:       "example.com",
				UseLocalHost: false,
			},
			url:               "https://test.example.com",
			expectedSubdomain: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ExtractSubdomain(tt.url)
			assert.Equal(t, tt.expectedSubdomain, result)
		})
	}
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Setup environment variables
	oldVars := map[string]string{
		"PORTR_SSH_PORT":               os.Getenv("PORTR_SSH_PORT"),
		"PORTR_PROXY_PORT":             os.Getenv("PORTR_PROXY_PORT"),
		"PORTR_DOMAIN":                 os.Getenv("PORTR_DOMAIN"),
		"PORTR_TUNNEL_USE_LOCALHOST":   os.Getenv("PORTR_TUNNEL_USE_LOCALHOST"),
		"PORTR_TUNNEL_DEBUG":           os.Getenv("PORTR_TUNNEL_DEBUG"),
		"PORTR_DB_URL":                 os.Getenv("PORTR_DB_URL"),
	}
	defer func() {
		for key, value := range oldVars {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Set test values
	os.Setenv("PORTR_SSH_PORT", "3333")
	os.Setenv("PORTR_PROXY_PORT", "9999")
	os.Setenv("PORTR_DOMAIN", "test.example.com")
	os.Setenv("PORTR_TUNNEL_USE_LOCALHOST", "true")
	os.Setenv("PORTR_TUNNEL_DEBUG", "true")
	os.Setenv("PORTR_DB_URL", "sqlite:///test.db")

	config := Load("")

	assert.Equal(t, 3333, config.Ssh.Port)
	assert.Equal(t, 9999, config.Proxy.Port)
	assert.Equal(t, "test.example.com", config.Domain)
	assert.True(t, config.UseLocalHost)
	assert.True(t, config.Debug)
	assert.Equal(t, "sqlite:///test.db", config.Database.Url)
	assert.Equal(t, "sqlite", config.Database.Driver)
}

func TestLoad_WithDefaults(t *testing.T) {
	// Clear relevant environment variables
	oldVars := map[string]string{
		"PORTR_SSH_PORT":               os.Getenv("PORTR_SSH_PORT"),
		"PORTR_PROXY_PORT":             os.Getenv("PORTR_PROXY_PORT"),
		"PORTR_DOMAIN":                 os.Getenv("PORTR_DOMAIN"),
		"PORTR_TUNNEL_USE_LOCALHOST":   os.Getenv("PORTR_TUNNEL_USE_LOCALHOST"),
		"PORTR_TUNNEL_DEBUG":           os.Getenv("PORTR_TUNNEL_DEBUG"),
		"PORTR_DB_URL":                 os.Getenv("PORTR_DB_URL"),
	}
	defer func() {
		for key, value := range oldVars {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	os.Unsetenv("PORTR_SSH_PORT")
	os.Unsetenv("PORTR_PROXY_PORT")
	os.Unsetenv("PORTR_DOMAIN")
	os.Unsetenv("PORTR_TUNNEL_USE_LOCALHOST")
	os.Unsetenv("PORTR_TUNNEL_DEBUG")
	os.Setenv("PORTR_DB_URL", "postgres://localhost/test")

	config := Load("")

	assert.Equal(t, 2222, config.Ssh.Port)
	assert.Equal(t, 8001, config.Proxy.Port)
	assert.Equal(t, "localhost:8001", config.Domain)
	assert.False(t, config.UseLocalHost)
	assert.False(t, config.Debug)
	assert.Equal(t, "postgres://localhost/test", config.Database.Url)
	assert.Equal(t, "postgres", config.Database.Driver)
}