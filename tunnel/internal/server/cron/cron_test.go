package cron

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestCron(t *testing.T) (*Cron, *service.Service) {
	// Use in-memory SQLite for testing
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate tables
	err = gormDB.AutoMigrate(&db.Connection{}, &db.TeamUser{})
	require.NoError(t, err)

	dbConfig := &config.DatabaseConfig{
		Url:    ":memory:",
		Driver: "sqlite",
	}

	testDB := db.New(dbConfig)
	testDB.Conn = gormDB

	config := &config.Config{
		Domain:       "localhost:8001",
		UseLocalHost: true,
		Proxy: config.ProxyConfig{
			Host: "localhost",
			Port: 8001,
		},
	}

	service := service.New(testDB)
	cron := New(testDB, config, service)

	return cron, service
}

func createTestConnection(t *testing.T, dbConn *db.Db, connType, status string, subdomain *string, port *uint32) *db.Connection {
	teamUser := &db.TeamUser{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		SecretKey: "test-secret",
		Role:      "admin",
		TeamID:    1,
		UserID:    1,
	}

	err := dbConn.Conn.Create(teamUser).Error
	require.NoError(t, err)

	connection := &db.Connection{
		ID:          "test-conn-" + time.Now().Format("150405.000"),
		Type:        connType,
		Subdomain:   subdomain,
		Port:        port,
		Status:      status,
		CreatedAt:   time.Now(),
		CreatedByID: teamUser.ID,
		CreatedBy:   *teamUser,
	}

	err = dbConn.Conn.Create(connection).Error
	require.NoError(t, err)

	return connection
}

func TestNew(t *testing.T) {
	cron, service := setupTestCron(t)

	assert.NotNil(t, cron)
	assert.NotNil(t, cron.db)
	assert.NotNil(t, cron.config)
	assert.Equal(t, service, cron.service)
}

func TestCron_Start_And_Shutdown(t *testing.T) {
	cron, _ := setupTestCron(t)

	// Start cron jobs
	cron.Start()

	// Let it run for a short time
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	cron.Shutdown()

	// Verify shutdown was called (cancelFunc should be set)
	assert.NotNil(t, cron.cancelFunc)
}

func TestCron_PingHttpConnection_Success(t *testing.T) {
	// Create a test HTTP server that responds to ping requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if it's a ping request
		if r.Header.Get("X-Portr-Ping-Request") == "true" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cron, _ := setupTestCron(t)

	// Create a test connection
	subdomain := "test"
	connection := db.Connection{
		ID:        "test-conn-1",
		Type:      "http",
		Subdomain: &subdomain,
		Status:    "active",
	}

	// Mock the HttpTunnelUrl to return our test server URL
	cron.config.Domain = server.URL[7:] // Remove "http://" prefix
	cron.config.UseLocalHost = false

	err := cron.pingHttpConnection(connection)
	assert.NoError(t, err)
}

func TestCron_PingHttpConnection_InactiveTunnel(t *testing.T) {
	// Create a test HTTP server that responds with Portr error headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Portr-Error", "true")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cron, _ := setupTestCron(t)

	// Create a test connection
	subdomain := "test"
	connection := db.Connection{
		ID:        "test-conn-1",
		Type:      "http",
		Subdomain: &subdomain,
		Status:    "active",
	}

	// Mock the HttpTunnelUrl to return our test server URL
	cron.config.Domain = server.URL[7:] // Remove "http://" prefix
	cron.config.UseLocalHost = false

	err := cron.pingHttpConnection(connection)
	assert.Error(t, err)
	assert.Equal(t, ErrInactiveTunnel, err)
}

func TestCron_PingTcpConnection_Success(t *testing.T) {
	// Start a TCP listener
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	// Get the actual port
	addr := listener.Addr().(*net.TCPAddr)
	port := uint32(addr.Port)

	cron, _ := setupTestCron(t)

	// Create a test connection
	connection := db.Connection{
		ID:     "test-conn-1",
		Type:   "tcp",
		Port:   &port,
		Status: "active",
	}

	err = cron.pingTcpConnection(connection)
	assert.NoError(t, err)
}

func TestCron_PingTcpConnection_InactiveTunnel(t *testing.T) {
	cron, _ := setupTestCron(t)

	// Create a test connection with a port that's not listening
	port := uint32(99999)
	connection := db.Connection{
		ID:     "test-conn-1",
		Type:   "tcp",
		Port:   &port,
		Status: "active",
	}

	err := cron.pingTcpConnection(connection)
	assert.Error(t, err)
	assert.Equal(t, ErrInactiveTunnel, err)
}

func TestCron_PingActiveConnections_HttpSuccess(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cron, _ := setupTestCron(t)

	// Mock the HttpTunnelUrl to return our test server URL
	cron.config.Domain = server.URL[7:] // Remove "http://" prefix
	cron.config.UseLocalHost = false

	// Create an active HTTP connection
	subdomain := "test"
	_ = createTestConnection(t, cron.db, "http", "active", &subdomain, nil)

	// Run ping active connections
	ctx := context.Background()
	cron.pingActiveConnections(ctx)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Verify connection is still active (no error occurred)
	connections := cron.service.GetAllActiveConnections(ctx)
	assert.Len(t, connections, 1)
	assert.Equal(t, "active", connections[0].Status)
}

func TestCron_PingActiveConnections_HttpFailure(t *testing.T) {
	cron, _ := setupTestCron(t)

	// Create an active HTTP connection with non-existent subdomain
	subdomain := "nonexistent"
	connection := createTestConnection(t, cron.db, "http", "active", &subdomain, nil)

	// Run ping active connections
	ctx := context.Background()
	cron.pingActiveConnections(ctx)

	// Give some time for the goroutine to complete
	time.Sleep(200 * time.Millisecond)

	// Verify connection was marked as closed
	var updatedConnection db.Connection
	err := cron.db.Conn.First(&updatedConnection, "id = ?", connection.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "closed", updatedConnection.Status)
}

func TestCron_PingActiveConnections_TcpSuccess(t *testing.T) {
	// Start a TCP listener
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	// Get the actual port
	addr := listener.Addr().(*net.TCPAddr)
	port := uint32(addr.Port)

	cron, _ := setupTestCron(t)

	// Create an active TCP connection
	_ = createTestConnection(t, cron.db, "tcp", "active", nil, &port)

	// Run ping active connections
	ctx := context.Background()
	cron.pingActiveConnections(ctx)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Verify connection is still active
	connections := cron.service.GetAllActiveConnections(ctx)
	assert.Len(t, connections, 1)
	assert.Equal(t, "active", connections[0].Status)
}

func TestCron_PingActiveConnections_TcpFailure(t *testing.T) {
	cron, _ := setupTestCron(t)

	// Create an active TCP connection with non-listening port
	port := uint32(99999)
	connection := createTestConnection(t, cron.db, "tcp", "active", nil, &port)

	// Run ping active connections
	ctx := context.Background()
	cron.pingActiveConnections(ctx)

	// Give some time for the goroutine to complete
	time.Sleep(200 * time.Millisecond)

	// Verify connection was marked as closed
	var updatedConnection db.Connection
	err := cron.db.Conn.First(&updatedConnection, "id = ?", connection.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "closed", updatedConnection.Status)
}

func TestCron_PingActiveConnections_NoActiveConnections(t *testing.T) {
	cron, _ := setupTestCron(t)

	// Create only non-active connections
	subdomain := "test"
	_ = createTestConnection(t, cron.db, "http", "closed", &subdomain, nil)
	_ = createTestConnection(t, cron.db, "http", "reserved", &subdomain, nil)

	// Run ping active connections
	ctx := context.Background()
	cron.pingActiveConnections(ctx)

	// Should complete without errors (no active connections to ping)
	connections := cron.service.GetAllActiveConnections(ctx)
	assert.Len(t, connections, 0)
}

func TestCron_PingActiveConnections_MixedResults(t *testing.T) {
	// Start a TCP listener for one connection
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	workingPort := uint32(addr.Port)

	cron, _ := setupTestCron(t)

	// Create one working TCP connection
	workingConnection := createTestConnection(t, cron.db, "tcp", "active", nil, &workingPort)

	// Create one failing TCP connection
	failingPort := uint32(99999)
	failingConnection := createTestConnection(t, cron.db, "tcp", "active", nil, &failingPort)

	// Run ping active connections
	ctx := context.Background()
	cron.pingActiveConnections(ctx)

	// Give some time for the goroutines to complete
	time.Sleep(200 * time.Millisecond)

	// Check final states
	var workingConn, failingConn db.Connection

	err = cron.db.Conn.First(&workingConn, "id = ?", workingConnection.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "active", workingConn.Status)

	err = cron.db.Conn.First(&failingConn, "id = ?", failingConnection.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "closed", failingConn.Status)
}