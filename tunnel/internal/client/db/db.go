package db

import (
	"log"
	"os"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Db struct {
	Conn *gorm.DB
}

func New() *Db {
	homeDir, _ := os.UserHomeDir()
	db, err := gorm.Open(sqlite.Open(homeDir+"/.portr/db.sqlite"), &gorm.Config{})
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
	Url                string
	Method             string
	Headers            datatypes.JSON
	Body               []byte
	ResponseHeaders    datatypes.JSON
	ResponseBody       []byte
	ResponseStatusCode int
	LoggedAt           time.Time
}
