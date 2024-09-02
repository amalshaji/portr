package db

import (
	"log"
	"os"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Db struct {
	Conn *gorm.DB
}

func New(config *config.Config) *Db {
	homeDir, _ := os.UserHomeDir()

	gormConfig := &gorm.Config{
		SkipDefaultTransaction: true,
	}
	if !config.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(sqlite.Open(homeDir+"/.portr/db.sqlite"), gormConfig)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
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
