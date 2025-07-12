package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestConfig() *config.Config {
	return &config.Config{
		Proxy: config.ProxyConfig{
			Host: "localhost",
			Port: 8001,
		},
		Domain:       "localhost:8001",
		UseLocalHost: true,
		Debug:        false,
	}
}

func TestNew(t *testing.T) {
	cfg := createTestConfig()
	proxy := New(cfg)

	assert.NotNil(t, proxy)
	assert.Equal(t, cfg, proxy.config)
	assert.NotNil(t, proxy.routes)
	assert.NotNil(t, proxy.server)
	assert.Equal(t, ":8001", proxy.GetServerAddr())
}

func TestProxy_GetServerAddr(t *testing.T) {
	cfg := createTestConfig()
	cfg.Proxy.Port = 9999
	proxy := New(cfg)

	assert.Equal(t, ":9999", proxy.GetServerAddr())
}

func TestProxy_AddRoute(t *testing.T) {
	proxy := New(createTestConfig())

	err := proxy.AddRoute("test", "localhost:3000")
	assert.NoError(t, err)

	route, err := proxy.GetRoute("test")
	assert.NoError(t, err)
	assert.Equal(t, "localhost:3000", route)
}

func TestProxy_AddRoute_Duplicate(t *testing.T) {
	proxy := New(createTestConfig())

	err := proxy.AddRoute("test", "localhost:3000")
	assert.NoError(t, err)

	// Adding the same route again should fail
	err = proxy.AddRoute("test", "localhost:4000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "route already added")
}

func TestProxy_GetRoute_NotFound(t *testing.T) {
	proxy := New(createTestConfig())

	_, err := proxy.GetRoute("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "route not found")
}

func TestProxy_RemoveRoute(t *testing.T) {
	proxy := New(createTestConfig())

	// Add a route first
	err := proxy.AddRoute("test", "localhost:3000")
	assert.NoError(t, err)

	// Remove it
	err = proxy.RemoveRoute("test")
	assert.NoError(t, err)

	// Verify it's gone
	_, err = proxy.GetRoute("test")
	assert.Error(t, err)
}

func TestProxy_RemoveRoute_NotFound(t *testing.T) {
	proxy := New(createTestConfig())

	err := proxy.RemoveRoute("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "route not found")
}

func TestProxy_HandleRequest_Success(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from backend"))
	}))
	defer backend.Close()

	// Extract port from backend URL
	backendAddr := strings.TrimPrefix(backend.URL, "http://")

	proxy := New(createTestConfig())
	err := proxy.AddRoute("test", backendAddr)
	require.NoError(t, err)

	// Create a request to the proxy
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "test.localhost:8001"
	w := httptest.NewRecorder()

	proxy.handleRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello from backend", w.Body.String())
}

func TestProxy_HandleRequest_UnregisteredSubdomain(t *testing.T) {
	proxy := New(createTestConfig())

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "nonexistent.localhost:8001"
	w := httptest.NewRecorder()

	proxy.handleRequest(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "true", w.Header().Get("X-Portr-Error"))
	assert.Equal(t, "unregistered-subdomain", w.Header().Get("X-Portr-Error-Reason"))
}

func TestProxy_HandleRequest_ConnectionLost(t *testing.T) {
	proxy := New(createTestConfig())

	// Add a route to a non-existent backend
	err := proxy.AddRoute("test", "localhost:99999")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "test.localhost:8001"
	w := httptest.NewRecorder()

	proxy.handleRequest(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "true", w.Header().Get("X-Portr-Error"))
	assert.Equal(t, "connection-lost", w.Header().Get("X-Portr-Error-Reason"))

	// Route should be automatically removed after connection error
	_, err = proxy.GetRoute("test")
	assert.Error(t, err)
}

func TestProxy_ExtractSubdomain_Localhost(t *testing.T) {
	cfg := createTestConfig()
	cfg.UseLocalHost = true
	proxy := New(cfg)

	tests := []struct {
		host              string
		expectedSubdomain string
	}{
		{"test.localhost:8001", "test"},
		{"my-app.localhost:8001", "my-app"},
		{"complex-name.localhost:8001", "complex-name"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			subdomain := proxy.config.ExtractSubdomain("http://" + tt.host)
			assert.Equal(t, tt.expectedSubdomain, subdomain)
		})
	}
}

func TestProxy_ExtractSubdomain_Production(t *testing.T) {
	cfg := createTestConfig()
	cfg.UseLocalHost = false
	cfg.Domain = "example.com"
	proxy := New(cfg)

	tests := []struct {
		host              string
		expectedSubdomain string
	}{
		{"test.example.com", "test"},
		{"my-app.example.com", "my-app"},
		{"complex-name.example.com", "complex-name"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			subdomain := proxy.config.ExtractSubdomain("https://" + tt.host)
			assert.Equal(t, tt.expectedSubdomain, subdomain)
		})
	}
}

func TestProxy_ConcurrentRouteOperations(t *testing.T) {
	proxy := New(createTestConfig())

	// Test concurrent operations to ensure thread safety
	done := make(chan bool, 100)

	// Add routes concurrently
	for i := 0; i < 50; i++ {
		go func(i int) {
			subdomain := fmt.Sprintf("test%d", i)
			target := fmt.Sprintf("localhost:%d", 3000+i)
			err := proxy.AddRoute(subdomain, target)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Get routes concurrently
	for i := 0; i < 50; i++ {
		go func(i int) {
			subdomain := fmt.Sprintf("test%d", i)
			_, err := proxy.GetRoute(subdomain)
			// Error is expected initially, but should not panic
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Verify all routes were added
	for i := 0; i < 50; i++ {
		subdomain := fmt.Sprintf("test%d", i)
		expectedTarget := fmt.Sprintf("localhost:%d", 3000+i)
		
		target, err := proxy.GetRoute(subdomain)
		assert.NoError(t, err)
		assert.Equal(t, expectedTarget, target)
	}
}

func TestProxy_ErrorHandler(t *testing.T) {
	proxy := New(createTestConfig())

	// Add a route
	err := proxy.AddRoute("test", "localhost:3000")
	require.NoError(t, err)

	// Create a mock request and response
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "test.localhost:8001"
	w := httptest.NewRecorder()

	// Simulate an error
	testError := fmt.Errorf("connection refused")
	proxy.ErrHandle(w, req, testError)

	// Check that the route was removed
	_, err = proxy.GetRoute("test")
	assert.Error(t, err)

	// Check error response
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "true", w.Header().Get("X-Portr-Error"))
	assert.Equal(t, "connection-lost", w.Header().Get("X-Portr-Error-Reason"))
}