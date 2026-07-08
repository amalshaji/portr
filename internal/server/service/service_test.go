package service

import (
	"context"
	"testing"

	serverdb "github.com/amalshaji/portr/internal/server/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCloseAllActiveConnectionsReconcilesStartupState(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(&serverdb.Connection{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	connections := []serverdb.Connection{
		{ID: "stale-http", Type: "http", Status: "active"},
		{ID: "stale-tcp", Type: "tcp", Status: "active"},
		{ID: "reserved", Type: "http", Status: "reserved"},
	}
	if err := database.Create(&connections).Error; err != nil {
		t.Fatalf("create connections: %v", err)
	}

	service := New(&serverdb.Db{Conn: database})
	if err := service.CloseAllActiveConnections(context.Background()); err != nil {
		t.Fatalf("reconcile active connections: %v", err)
	}

	for _, id := range []string{"stale-http", "stale-tcp"} {
		var connection serverdb.Connection
		if err := database.First(&connection, "id = ?", id).Error; err != nil {
			t.Fatalf("load %s: %v", id, err)
		}
		if connection.Status != "closed" || connection.ClosedAt == nil {
			t.Fatalf("expected %s closed, status=%q closed_at=%v", id, connection.Status, connection.ClosedAt)
		}
	}

	var reserved serverdb.Connection
	if err := database.First(&reserved, "id = ?", "reserved").Error; err != nil {
		t.Fatalf("load reserved: %v", err)
	}
	if reserved.Status != "reserved" || reserved.ClosedAt != nil {
		t.Fatalf("reserved connection changed: status=%q closed_at=%v", reserved.Status, reserved.ClosedAt)
	}
}
