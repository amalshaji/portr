package service

import (
	"context"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *db.Db {
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

	return testDB
}

func createTestConnection(t *testing.T, dbConn *db.Db, status string) *db.Connection {
	teamUser := &db.TeamUser{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		SecretKey: "test-secret",
		Role:      "admin",
		TeamID:    1,
		UserID:    1,
	}

	err := dbConn.Conn.Create(teamUser).Error
	require.NoError(t, err)

	subdomain := "test"
	connection := &db.Connection{
		ID:          "test-conn-1",
		Type:        "http",
		Subdomain:   &subdomain,
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
	testDB := setupTestDB(t)
	service := New(testDB)

	assert.NotNil(t, service)
	assert.Equal(t, testDB, service.db)
}

func TestService_GetReservedConnectionById(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create a reserved connection
	connection := createTestConnection(t, testDB, "reserved")

	// Test successful retrieval
	result, err := service.GetReservedConnectionById(ctx, connection.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, connection.ID, result.ID)
	assert.Equal(t, "reserved", result.Status)
	assert.NotNil(t, result.CreatedBy)
	assert.Equal(t, "test-secret", result.CreatedBy.SecretKey)
}

func TestService_GetReservedConnectionById_NotFound(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Test non-existent connection
	_, err := service.GetReservedConnectionById(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

func TestService_GetReservedConnectionById_WrongStatus(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create a connection with non-reserved status
	connection := createTestConnection(t, testDB, "active")

	// Test retrieval should fail because status is not 'reserved'
	_, err := service.GetReservedConnectionById(ctx, connection.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

func TestService_AddPortToConnection(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create a reserved connection
	connection := createTestConnection(t, testDB, "reserved")

	// Add port to connection
	var port uint32 = 8080
	err := service.AddPortToConnection(ctx, connection.ID, port)
	assert.NoError(t, err)

	// Verify port was added
	var updatedConnection db.Connection
	err = testDB.Conn.First(&updatedConnection, "id = ?", connection.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, updatedConnection.Port)
	assert.Equal(t, port, *updatedConnection.Port)
}

func TestService_AddPortToConnection_NotFound(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Try to add port to non-existent connection
	var port uint32 = 8080
	err := service.AddPortToConnection(ctx, "non-existent", port)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

func TestService_MarkConnectionAsActive(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create a reserved connection
	connection := createTestConnection(t, testDB, "reserved")

	// Mark as active
	err := service.MarkConnectionAsActive(ctx, connection.ID)
	assert.NoError(t, err)

	// Verify status was updated
	var updatedConnection db.Connection
	err = testDB.Conn.First(&updatedConnection, "id = ?", connection.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "active", updatedConnection.Status)
	assert.NotNil(t, updatedConnection.StartedAt)
	assert.True(t, time.Since(*updatedConnection.StartedAt) < time.Minute)
}

func TestService_MarkConnectionAsClosed(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create an active connection
	connection := createTestConnection(t, testDB, "active")

	// Mark as closed
	err := service.MarkConnectionAsClosed(ctx, connection.ID)
	assert.NoError(t, err)

	// Verify status was updated
	var updatedConnection db.Connection
	err = testDB.Conn.First(&updatedConnection, "id = ?", connection.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "closed", updatedConnection.Status)
	assert.NotNil(t, updatedConnection.ClosedAt)
	assert.True(t, time.Since(*updatedConnection.ClosedAt) < time.Minute)
}

func TestService_GetAllActiveConnections(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create multiple connections with different statuses
	_ = createTestConnection(t, testDB, "active")
	_ = createTestConnection(t, testDB, "closed")
	_ = createTestConnection(t, testDB, "reserved")

	// Create another active connection with different ID
	teamUser := &db.TeamUser{
		ID:        2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		SecretKey: "test-secret-2",
		Role:      "admin",
		TeamID:    1,
		UserID:    2,
	}
	err := testDB.Conn.Create(teamUser).Error
	require.NoError(t, err)

	subdomain2 := "test2"
	activeConnection2 := &db.Connection{
		ID:          "test-conn-2",
		Type:        "http",
		Subdomain:   &subdomain2,
		Status:      "active",
		CreatedAt:   time.Now(),
		CreatedByID: teamUser.ID,
	}
	err = testDB.Conn.Create(activeConnection2).Error
	require.NoError(t, err)

	// Get all active connections
	activeConnections := service.GetAllActiveConnections(ctx)

	// Should return only the active connections
	assert.Len(t, activeConnections, 2)
	for _, conn := range activeConnections {
		assert.Equal(t, "active", conn.Status)
	}
}

func TestService_GetAllActiveConnections_Empty(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create only non-active connections
	_ = createTestConnection(t, testDB, "closed")
	_ = createTestConnection(t, testDB, "reserved")

	// Get all active connections
	activeConnections := service.GetAllActiveConnections(ctx)

	// Should return empty slice
	assert.Len(t, activeConnections, 0)
}

func TestService_ConnectionLifecycle(t *testing.T) {
	testDB := setupTestDB(t)
	service := New(testDB)
	ctx := context.Background()

	// Create a reserved connection
	connection := createTestConnection(t, testDB, "reserved")

	// 1. Add port to connection
	var port uint32 = 8080
	err := service.AddPortToConnection(ctx, connection.ID, port)
	assert.NoError(t, err)

	// 2. Mark as active
	err = service.MarkConnectionAsActive(ctx, connection.ID)
	assert.NoError(t, err)

	// 3. Verify it appears in active connections
	activeConnections := service.GetAllActiveConnections(ctx)
	assert.Len(t, activeConnections, 1)
	assert.Equal(t, connection.ID, activeConnections[0].ID)

	// 4. Mark as closed
	err = service.MarkConnectionAsClosed(ctx, connection.ID)
	assert.NoError(t, err)

	// 5. Verify it no longer appears in active connections
	activeConnections = service.GetAllActiveConnections(ctx)
	assert.Len(t, activeConnections, 0)

	// 6. Verify final state
	var finalConnection db.Connection
	err = testDB.Conn.First(&finalConnection, "id = ?", connection.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "closed", finalConnection.Status)
	assert.NotNil(t, finalConnection.Port)
	assert.Equal(t, port, *finalConnection.Port)
	assert.NotNil(t, finalConnection.StartedAt)
	assert.NotNil(t, finalConnection.ClosedAt)
}