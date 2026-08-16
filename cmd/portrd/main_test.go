package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestMigrationsUpgradeFromPreviousRelease(t *testing.T) {
	const migrationDir = "../../migrations/sqlite"
	previousMigrations := []string{
		"20240101000000_init.sql",
		"20260707000100_unique_active_subdomain.sql",
		"20260710000100_reserved_subdomains.sql",
	}

	baselineDir := t.TempDir()
	for _, name := range previousMigrations {
		contents, err := os.ReadFile(filepath.Join(migrationDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baselineDir, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	database, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "portr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(database, baselineDir); err != nil {
		t.Fatalf("apply previous release migrations: %v", err)
	}
	if err := goose.Up(database, migrationDir); err != nil {
		t.Fatalf("upgrade from previous release: %v", err)
	}

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM auto_signup_settings").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one auto-signup settings row, got %d", count)
	}
}

func TestMembershipDedupeMigrationPreservesDependents(t *testing.T) {
	const migrationDir = "../../migrations/sqlite"
	beforeDedupe := []string{
		"20240101000000_init.sql",
		"20260707000100_unique_active_subdomain.sql",
		"20260710000100_reserved_subdomains.sql",
		"20260804000100_add_auto_signup_settings.sql",
	}

	baselineDir := t.TempDir()
	for _, name := range beforeDedupe {
		contents, err := os.ReadFile(filepath.Join(migrationDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baselineDir, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	database, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "portr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	// Cascades must be live so the test proves the migration does not rely on
	// them being off.
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(database, baselineDir); err != nil {
		t.Fatalf("apply pre-dedupe migrations: %v", err)
	}

	// A deployment with duplicate memberships, where connections and a
	// subdomain reservation hang off the duplicate rows.
	seed := []string{
		`INSERT INTO "user" (email) VALUES ('dup@example.com'), ('other@example.com')`,
		`INSERT INTO "team" (name, slug) VALUES ('team', 'team')`,
		`INSERT INTO "team_users" (id, secret_key, role, user_id, team_id) VALUES
			(1, 'sk-1', 'admin', 1, 1),
			(2, 'sk-2', 'admin', 1, 1),
			(3, 'sk-3', 'member', 2, 1),
			(4, 'sk-4', 'admin', 1, 1)`,
		`INSERT INTO "connection" (id, type, subdomain, status, created_by_id, team_id) VALUES
			('conn-dup-2', 'http', 'dup-two', 'closed', 2, 1),
			('conn-dup-4', 'http', 'dup-four', 'closed', 4, 1),
			('conn-keep', 'http', 'keep', 'closed', 1, 1),
			('conn-other', 'http', 'other', 'closed', 3, 1)`,
		`INSERT INTO "subdomain_reservation" (subdomain, team_user_id) VALUES
			('resv-dup', 2),
			('resv-other', 3)`,
	}
	for _, statement := range seed {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed fixture: %v (%s)", err, statement)
		}
	}

	if err := goose.Up(database, migrationDir); err != nil {
		t.Fatalf("upgrade with duplicate memberships: %v", err)
	}

	var memberships int
	if err := database.QueryRow(`SELECT COUNT(*) FROM "team_users"`).Scan(&memberships); err != nil {
		t.Fatal(err)
	}
	if memberships != 2 {
		t.Fatalf("expected the duplicates to collapse to 2 memberships, got %d", memberships)
	}

	rows := map[string]int{}
	result, err := database.Query(`SELECT id, created_by_id FROM "connection"`)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Close()
	for result.Next() {
		var id string
		var createdBy int
		if err := result.Scan(&id, &createdBy); err != nil {
			t.Fatal(err)
		}
		rows[id] = createdBy
	}
	if err := result.Err(); err != nil {
		t.Fatal(err)
	}
	expected := map[string]int{"conn-dup-2": 1, "conn-dup-4": 1, "conn-keep": 1, "conn-other": 3}
	if len(rows) != len(expected) {
		t.Fatalf("expected %d connections to survive, got %v", len(expected), rows)
	}
	for id, owner := range expected {
		if rows[id] != owner {
			t.Fatalf("expected connection %s to be owned by membership %d, got %d", id, owner, rows[id])
		}
	}

	var reservationOwner int
	if err := database.QueryRow(`SELECT team_user_id FROM "subdomain_reservation" WHERE subdomain = 'resv-dup'`).Scan(&reservationOwner); err != nil {
		t.Fatalf("the reservation owned by a duplicate membership must survive: %v", err)
	}
	if reservationOwner != 1 {
		t.Fatalf("expected the reservation to move to membership 1, got %d", reservationOwner)
	}

	if _, err := database.Exec(`INSERT INTO "team_users" (secret_key, role, user_id, team_id) VALUES ('sk-clash', 'member', 1, 1)`); err == nil {
		t.Fatal("expected the unique index to reject a duplicate membership")
	}
}
