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

func defaultSettings() map[string]string {
	return map[string]string{
		"signup_requires_invite":             "true",
		"allow_random_user_signup":           "false",
		"random_user_signup_allowed_domains": "",
	}
}

func (d *Db) Connect() {
	var err error

	d.Conn, err = gorm.Open(sqlite.Open("./data/db.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	if err := d.Conn.AutoMigrate(
		&User{},
		&Invite{},
		&Session{},
		&OAuthState{},
		&Connection{},
		&Settings{},
	); err != nil {
		log.Fatal(err)
	}

	// populate default settings
	for name, value := range defaultSettings() {
		var count int64
		d.Conn.Model(&Settings{}).Where("name = ?", name).Count(&count)
		if count == 0 {
			d.Conn.Create(&Settings{Name: name, Value: value})
		}
	}
}
