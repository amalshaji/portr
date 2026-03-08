package db

import (
	"os"
	"time"

	"github.com/charmbracelet/log"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Db struct {
	Conn *gorm.DB
}

var SQLITE_PRAGMAS = []string{
	`PRAGMA foreign_keys = ON;`,
	`PRAGMA journal_mode = WAL;`,
	`PRAGMA synchronous = NORMAL;`,
	`PRAGMA busy_timeout = 5000;`,
	`PRAGMA temp_store = MEMORY;`,
	`PRAGMA mmap_size = 134217728;`,
	`PRAGMA journal_size_limit = 67108864;`,
	`PRAGMA cache_size = 2000;`,
}

func New(config *config.Config) *Db {
	homeDir, _ := os.UserHomeDir()

	gormConfig := &gorm.Config{
		SkipDefaultTransaction: true,
		TranslateError:         true,
		PrepareStmt:            true,
	}
	if !config.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(sqlite.Open(homeDir+"/.portr/db.sqlite"), gormConfig)
	if err != nil {
		log.Fatal("Failed to connect database", "error", err)
	}

	conn, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database connection", "error", err)
	}

	for _, pragma := range SQLITE_PRAGMAS {
		_, err = conn.Exec(pragma)
		if err != nil {
			log.Fatal("Failed to set pragma", "error", err)
		}
	}

	db.AutoMigrate(&Request{})

	return &Db{
		Conn: db,
	}
}

type Request struct {
	ID                 string `gorm:"primaryKey"`
	Subdomain          string
	Localport          int
	Host               string
	Url                string
	Method             string
	Headers            datatypes.JSON
	Body               []byte
	ResponseHeaders    datatypes.JSON
	ResponseBody       []byte
	ResponseStatusCode int
	LoggedAt           time.Time
	IsReplayed         bool
	ParentID           string
}

func (d *Db) DeleteRequestsOlderThan(cutoff time.Time) (int64, error) {
	result := d.Conn.Where("logged_at < ?", cutoff).Delete(&Request{})
	return result.RowsAffected, result.Error
}
