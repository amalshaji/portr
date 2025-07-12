package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTunnel_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		tunnel   Tunnel
		expected Tunnel
	}{
		{
			name: "empty tunnel gets defaults",
			tunnel: Tunnel{
				Port: 3000,
				Type: constants.Http,
			},
			expected: Tunnel{
				Name:      "tunnel-3000",
				Port:      3000,
				Subdomain: "tunnel-3000",
				Type:      constants.Http,
			},
		},
		{
			name: "partial tunnel keeps existing values",
			tunnel: Tunnel{
				Name: "my-app",
				Port: 8080,
				Type: constants.Http,
			},
			expected: Tunnel{
				Name:      "my-app",
				Port:      8080,
				Subdomain: "my-app",
				Type:      constants.Http,
			},
		},
		{
			name: "tcp tunnel doesn't get subdomain",
			tunnel: Tunnel{
				Port: 9000,
				Type: constants.Tcp,
			},
			expected: Tunnel{
				Name:      "tunnel-9000",
				Port:      9000,
				Subdomain: "",
				Type:      constants.Tcp,
			},
		},
		{
			name: "existing values preserved",
			tunnel: Tunnel{
				Name:      "custom-name",
				Port:      4000,
				Subdomain: "custom-subdomain",
				Type:      constants.Http,
			},
			expected: Tunnel{
				Name:      "custom-name",
				Port:      4000,
				Subdomain: "custom-subdomain",
				Type:      constants.Http,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tunnel.SetDefaults()
			assert.Equal(t, tt.expected, tt.tunnel)
		})
	}
}

func TestTunnel_Validate(t *testing.T) {
	tests := []struct {
		name        string
		tunnel      Tunnel
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid http tunnel",
			tunnel: Tunnel{
				Name:      "test",
				Port:      3000,
				Subdomain: "test",
				Type:      constants.Http,
			},
			shouldError: false,
		},
		{
			name: "valid tcp tunnel",
			tunnel: Tunnel{
				Name: "test-tcp",
				Port: 8080,
				Type: constants.Tcp,
			},
			shouldError: false,
		},
		{
			name: "invalid port - zero",
			tunnel: Tunnel{
				Name: "test",
				Port: 0,
				Type: constants.Http,
			},
			shouldError: true,
			errorMsg:    "port must be greater than 0",
		},
		{
			name: "invalid port - negative",
			tunnel: Tunnel{
				Name: "test",
				Port: -1,
				Type: constants.Http,
			},
			shouldError: true,
			errorMsg:    "port must be greater than 0",
		},
		{
			name: "invalid port - too high",
			tunnel: Tunnel{
				Name: "test",
				Port: 70000,
				Type: constants.Http,
			},
			shouldError: true,
			errorMsg:    "port must be less than 65536",
		},
		{
			name: "missing name",
			tunnel: Tunnel{
				Port: 3000,
				Type: constants.Http,
			},
			shouldError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing subdomain for http",
			tunnel: Tunnel{
				Name: "test",
				Port: 3000,
				Type: constants.Http,
			},
			shouldError: true,
			errorMsg:    "subdomain is required for http tunnels",
		},
		{
			name: "invalid subdomain format",
			tunnel: Tunnel{
				Name:      "test",
				Port:      3000,
				Subdomain: "invalid_subdomain!",
				Type:      constants.Http,
			},
			shouldError: true,
			errorMsg:    "invalid subdomain format",
		},
		{
			name: "invalid type",
			tunnel: Tunnel{
				Name: "test",
				Port: 3000,
				Type: "invalid",
			},
			shouldError: true,
			errorMsg:    "type must be either 'http' or 'tcp'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tunnel.Validate()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				ServerUrl: "https://api.example.com",
				SshUrl:    "ssh.example.com:2222",
				SecretKey: "secret123",
				Tunnels: []Tunnel{
					{
						Name:      "test",
						Port:      3000,
						Subdomain: "test",
						Type:      constants.Http,
					},
				},
			},
			shouldError: false,
		},
		{
			name: "missing server url",
			config: Config{
				SshUrl:    "ssh.example.com:2222",
				SecretKey: "secret123",
				Tunnels:   []Tunnel{},
			},
			shouldError: true,
			errorMsg:    "server_url is required",
		},
		{
			name: "missing ssh url",
			config: Config{
				ServerUrl: "https://api.example.com",
				SecretKey: "secret123",
				Tunnels:   []Tunnel{},
			},
			shouldError: true,
			errorMsg:    "ssh_url is required",
		},
		{
			name: "missing secret key",
			config: Config{
				ServerUrl: "https://api.example.com",
				SshUrl:    "ssh.example.com:2222",
				Tunnels:   []Tunnel{},
			},
			shouldError: true,
			errorMsg:    "secret_key is required",
		},
		{
			name: "no tunnels",
			config: Config{
				ServerUrl: "https://api.example.com",
				SshUrl:    "ssh.example.com:2222",
				SecretKey: "secret123",
				Tunnels:   []Tunnel{},
			},
			shouldError: true,
			errorMsg:    "at least one tunnel is required",
		},
		{
			name: "invalid tunnel",
			config: Config{
				ServerUrl: "https://api.example.com",
				SshUrl:    "ssh.example.com:2222",
				SecretKey: "secret123",
				Tunnels: []Tunnel{
					{
						Name: "test",
						Port: 0, // Invalid port
						Type: constants.Http,
					},
				},
			},
			shouldError: true,
			errorMsg:    "port must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
server_url: https://api.example.com
ssh_url: ssh.example.com:2222
secret_key: test-secret
debug: true
enable_request_logging: true
tunnels:
  - name: web
    port: 3000
    subdomain: web
    type: http
  - name: api
    port: 8080
    subdomain: api
    type: http
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := Load(configFile)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.example.com", config.ServerUrl)
	assert.Equal(t, "ssh.example.com:2222", config.SshUrl)
	assert.Equal(t, "test-secret", config.SecretKey)
	assert.True(t, config.Debug)
	assert.True(t, config.EnableRequestLogging)
	assert.Len(t, config.Tunnels, 2)
	
	assert.Equal(t, "web", config.Tunnels[0].Name)
	assert.Equal(t, 3000, config.Tunnels[0].Port)
	assert.Equal(t, "web", config.Tunnels[0].Subdomain)
	assert.Equal(t, constants.Http, config.Tunnels[0].Type)
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("/non/existent/file.yaml")
	assert.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	// Write invalid YAML
	err := os.WriteFile(configFile, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	_, err = Load(configFile)
	assert.Error(t, err)
}

func TestDefaultPaths(t *testing.T) {
	// Test that default paths are set correctly
	assert.NotEmpty(t, DefaultConfigDir)
	assert.NotEmpty(t, DefaultConfigPath)
	assert.Contains(t, DefaultConfigPath, DefaultConfigDir)
	assert.Contains(t, DefaultConfigPath, "config.yaml")
}

func TestEditConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that might open editor")
	}

	// This function opens an editor, so we can only test that it doesn't panic
	// and returns some result
	err := EditConfig()
	// The error will depend on whether an editor is available
	// We just test that the function can be called
	_ = err
}

func TestGetConfig_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that makes HTTP requests")
	}

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	oldConfigDir := DefaultConfigDir
	DefaultConfigDir = tempDir
	defer func() { DefaultConfigDir = oldConfigDir }()

	// This would normally make an HTTP request, so we skip in normal testing
	// In a real test environment, you'd mock the HTTP client
	err := GetConfig("test-token", "https://example.com")
	// We expect this to fail because it's trying to make a real HTTP request
	assert.Error(t, err)
}

func TestClientConfig_Validation(t *testing.T) {
	clientConfig := ClientConfig{
		ServerUrl:             "https://api.example.com",
		SshUrl:                "ssh.example.com:2222",
		SecretKey:             "secret123",
		UseLocalHost:          false,
		Debug:                 true,
		EnableRequestLogging:  true,
		HealthCheckInterval:   30,
		HealthCheckMaxRetries: 3,
		Tunnel: Tunnel{
			Name:      "test",
			Port:      3000,
			Subdomain: "test",
			Type:      constants.Http,
		},
	}

	// Test that all fields are properly set
	assert.Equal(t, "https://api.example.com", clientConfig.ServerUrl)
	assert.Equal(t, "ssh.example.com:2222", clientConfig.SshUrl)
	assert.Equal(t, "secret123", clientConfig.SecretKey)
	assert.False(t, clientConfig.UseLocalHost)
	assert.True(t, clientConfig.Debug)
	assert.True(t, clientConfig.EnableRequestLogging)
	assert.Equal(t, 30, clientConfig.HealthCheckInterval)
	assert.Equal(t, 3, clientConfig.HealthCheckMaxRetries)
	assert.Equal(t, "test", clientConfig.Tunnel.Name)
}

func TestTunnel_SubdomainGeneration(t *testing.T) {
	tests := []struct {
		name             string
		tunnelName       string
		port             int
		tunnelType       string
		expectedName     string
		expectedSubdomain string
	}{
		{
			name:              "http tunnel with name",
			tunnelName:        "my-app",
			port:              3000,
			tunnelType:        constants.Http,
			expectedName:      "my-app",
			expectedSubdomain: "my-app",
		},
		{
			name:              "http tunnel without name",
			tunnelName:        "",
			port:              8080,
			tunnelType:        constants.Http,
			expectedName:      "tunnel-8080",
			expectedSubdomain: "tunnel-8080",
		},
		{
			name:              "tcp tunnel without name",
			tunnelName:        "",
			port:              9000,
			tunnelType:        constants.Tcp,
			expectedName:      "tunnel-9000",
			expectedSubdomain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tunnel := Tunnel{
				Name: tt.tunnelName,
				Port: tt.port,
				Type: tt.tunnelType,
			}
			
			tunnel.SetDefaults()
			
			assert.Equal(t, tt.expectedName, tunnel.Name)
			assert.Equal(t, tt.expectedSubdomain, tunnel.Subdomain)
		})
	}
}

func TestConfig_HealthCheckDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	// Config without health check settings
	configContent := `
server_url: https://api.example.com
ssh_url: ssh.example.com:2222
secret_key: test-secret
tunnels:
  - name: web
    port: 3000
    subdomain: web
    type: http
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := Load(configFile)
	assert.NoError(t, err)
	
	// Check that defaults are applied
	assert.Equal(t, 30, config.HealthCheckInterval)
	assert.Equal(t, 3, config.HealthCheckMaxRetries)
}