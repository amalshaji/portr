package db

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Db struct {
	Conn *gorm.DB
}

func New() *Db {
	return &Db{}
}

func (d *Db) Connect() {
	var err error

	d.Conn, err = gorm.Open(sqlite.Open("./data/db.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	if err := d.Conn.AutoMigrate(&User{}, &Invite{}, &Session{}, &OAuthState{}, &Connection{}); err != nil {
		log.Fatal(err)
	}
}
