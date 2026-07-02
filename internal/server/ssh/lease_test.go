package sshd

import (
	"testing"

	serverconfig "github.com/amalshaji/portr/internal/server/config"
	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newLeaseTestServer(t *testing.T) (*SshServer, *gorm.DB, *fakeSSHContext) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(&serverdb.TeamUser{}, &serverdb.Connection{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	teamUser := serverdb.TeamUser{SecretKey: "secret"}
	if err := database.Create(&teamUser).Error; err != nil {
		t.Fatalf("create team user: %v", err)
	}
	subdomain := "pooled"
	connection := serverdb.Connection{
		ID:          "connection",
		Type:        "http",
		Subdomain:   &subdomain,
		Status:      "reserved",
		CreatedByID: teamUser.ID,
	}
	if err := database.Create(&connection).Error; err != nil {
		t.Fatalf("create connection: %v", err)
	}

	proxyServer := proxy.New(&serverconfig.Config{Domain: "example.com"})
	sshServer := New(
		&serverconfig.SshConfig{},
		proxyServer,
		service.New(&serverdb.Db{Conn: database}),
	)
	ctx, cancel := newFakeSSHContext(t)
	t.Cleanup(cancel)
	return sshServer, database, ctx
}

func TestPooledConnectionClosesOnlyAfterLastForward(t *testing.T) {
	server, database, ctx := newLeaseTestServer(t)
	if err := server.activateForward(ctx, "127.0.0.1", 20001); err != nil {
		t.Fatalf("activate first forward: %v", err)
	}
	if err := server.activateForward(ctx, "127.0.0.1", 20002); err != nil {
		t.Fatalf("activate second forward: %v", err)
	}

	server.closeForward(ctx, "127.0.0.1", 20001)
	var connection serverdb.Connection
	if err := database.First(&connection, "id = ?", "connection").Error; err != nil {
		t.Fatalf("load connection: %v", err)
	}
	if connection.Status != "active" {
		t.Fatalf("expected pool to remain active, got %q", connection.Status)
	}

	server.closeForward(ctx, "127.0.0.1", 20002)
	if err := database.First(&connection, "id = ?", "connection").Error; err != nil {
		t.Fatalf("reload connection: %v", err)
	}
	if connection.Status != "closed" || connection.ClosedAt == nil {
		t.Fatalf("expected pool to close after last lease, status=%q closed_at=%v", connection.Status, connection.ClosedAt)
	}
}

func TestReactivatedConnectionClearsClosedTimestamp(t *testing.T) {
	server, database, ctx := newLeaseTestServer(t)
	if err := server.activateForward(ctx, "127.0.0.1", 20001); err != nil {
		t.Fatalf("activate forward: %v", err)
	}
	server.closeForward(ctx, "127.0.0.1", 20001)
	if err := server.activateForward(ctx, "127.0.0.1", 20002); err != nil {
		t.Fatalf("reactivate forward: %v", err)
	}

	var connection serverdb.Connection
	if err := database.First(&connection, "id = ?", "connection").Error; err != nil {
		t.Fatalf("load connection: %v", err)
	}
	if connection.Status != "active" || connection.ClosedAt != nil {
		t.Fatalf("expected clean active state, status=%q closed_at=%v", connection.Status, connection.ClosedAt)
	}
}
