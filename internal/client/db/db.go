package db

import (
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
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
	ID              string `gorm:"primaryKey"`
	Subdomain       string
	Localport       int
	Url             string
	Method          string
	Headers         datatypes.JSON
	Body            []byte
	ResponseStatus  int
	ResponseHeaders datatypes.JSON
	ResponseBody    []byte
}
