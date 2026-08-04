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
