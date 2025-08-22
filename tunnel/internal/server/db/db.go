package db

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalshaji/portr/internal/server/config"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Db struct {
	Conn   *gorm.DB
	config *config.DatabaseConfig
}

func New(config *config.DatabaseConfig) *Db {
	return &Db{
		config: config,
	}
}

func (d *Db) Connect() {
	var err error

	switch d.config.Driver {
	case "sqlite3", "sqlite":
		// Extract the path from the URL for SQLite
		dbPath := d.config.Url
		if strings.Contains(dbPath, "://") {
			parts := strings.Split(dbPath, "://")
			if len(parts) > 1 {
				dbPath = parts[1]
			}
		}
		
		// Ensure the directory exists for SQLite database
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Fatalf("failed to create database directory: %v", err)
		}
		
		d.Conn, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	case "postgres", "postgresql":
		d.Conn, err = gorm.Open(postgres.Open(d.config.Url), &gorm.Config{})
	default:
		log.Fatalf("unsupported database driver: %s", d.config.Driver)
	}

	if err != nil {
		log.Fatal(err)
	}
}
