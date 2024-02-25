package db

import (
	"log"

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
		d.Conn, err = gorm.Open(sqlite.Open(d.config.Url), &gorm.Config{})
	case "postgres", "postgresql":
		d.Conn, err = gorm.Open(postgres.Open(d.config.Url), &gorm.Config{})
	default:
		log.Fatalf("unsupported database driver: %s", d.config.Driver)
	}

	if err != nil {
		log.Fatal(err)
	}
}
