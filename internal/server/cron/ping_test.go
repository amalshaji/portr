package cron

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPingActiveConnectionsIsBoundedAndIsolatesFailures(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(&serverdb.Connection{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	connections := make([]serverdb.Connection, 0, 42)
	for index := 0; index < 40; index++ {
		connections = append(connections, serverdb.Connection{ID: fmt.Sprintf("healthy-%d", index), Type: "http", Status: "active"})
	}
	connections = append(connections,
		serverdb.Connection{ID: "inactive", Type: "http", Status: "active"},
		serverdb.Connection{ID: "transient", Type: "http", Status: "active"},
	)
	if err := database.Create(&connections).Error; err != nil {
		t.Fatalf("create connections: %v", err)
	}

	var current atomic.Int32
	var maximum atomic.Int32
	cron := &Cron{
		service: service.New(&serverdb.Db{Conn: database}),
	}
	probe := func(_ context.Context, connection serverdb.Connection) error {
		active := current.Add(1)
		defer current.Add(-1)
		for {
			observed := maximum.Load()
			if active <= observed || maximum.CompareAndSwap(observed, active) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		switch connection.ID {
		case "inactive":
			return ErrInactiveTunnel
		case "transient":
			return errors.New("temporary network failure")
		default:
			return nil
		}
	}

	cron.pingActiveConnectionsWithProbe(context.Background(), probe)
	if got := maximum.Load(); got > maxConcurrentPings {
		t.Fatalf("expected at most %d concurrent probes, got %d", maxConcurrentPings, got)
	}

	var inactive serverdb.Connection
	if err := database.First(&inactive, "id = ?", "inactive").Error; err != nil {
		t.Fatalf("load inactive connection: %v", err)
	}
	if inactive.Status != "closed" {
		t.Fatalf("expected inactive connection to close, got %q", inactive.Status)
	}
	var transient serverdb.Connection
	if err := database.First(&transient, "id = ?", "transient").Error; err != nil {
		t.Fatalf("load transient connection: %v", err)
	}
	if transient.Status != "active" {
		t.Fatalf("expected transient failure to remain active, got %q", transient.Status)
	}
}

func TestPingRejectsMissingRouteMetadata(t *testing.T) {
	cron := &Cron{}
	if !errors.Is(cron.pingHttpConnection(context.Background(), serverdb.Connection{}), ErrInactiveTunnel) {
		t.Fatal("missing HTTP subdomain should be inactive")
	}
	if !errors.Is(cron.pingTcpConnection(context.Background(), serverdb.Connection{}), ErrInactiveTunnel) {
		t.Fatal("missing TCP port should be inactive")
	}
}
