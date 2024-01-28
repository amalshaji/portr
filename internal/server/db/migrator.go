package db

import (
	"embed"
	"net/url"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
	"github.com/amalshaji/portr/internal/server/config"
)

//go:embed migrations/*.sql
var fs embed.FS

type Migrator struct {
	db     *Db
	config *config.DatabaseConfig
}

func NewMigrator(db *Db, config *config.DatabaseConfig) *Migrator {
	return &Migrator{db: db, config: config}
}

func (m *Migrator) Migrate() error {
	// dbmate requires it in this format
	dbUrl, _ := url.Parse(m.config.Driver + ":" + m.config.Url)
	_db := dbmate.New(dbUrl)
	_db.FS = fs
	_db.MigrationsDir = []string{"./migrations"}
	_db.SchemaFile = "./internal/server/db/schema.sql"

	if err := _db.CreateAndMigrate(); err != nil {
		return err
	}

	return nil
}
