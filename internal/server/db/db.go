package db

import (
	"log"
	"time"

	"github.com/amalshaji/localport/internal/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Db struct {
	Conn *gorm.DB
}

func New() *Db {
	return &Db{}
}

var (
	DefaultUserInviteEmailTemplate = utils.Trim(`Hi {{email}}, you have been invited to join team {{teamName}} on LocalPort. You can now signup using your GitHub account.

{{appUrl}}`)
)

func (d *Db) Connect() {
	var err error

	d.Conn, err = gorm.Open(sqlite.Open("./data/db.sqlite"), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := d.Conn.AutoMigrate(
		&Team{},
		&User{},
		&TeamUser{},
		&Invite{},
		&Session{},
		&Connection{},
		&Settings{},
	); err != nil {
		log.Fatal(err)
	}

	// Populate/update default settings
	var settings Settings
	result := d.Conn.First(&settings)
	if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
		settings.UserInviteEmailTemplate = DefaultUserInviteEmailTemplate

		result := d.Conn.Save(&settings)
		if result.Error != nil {
			log.Fatal(result.Error)
		}
	}

}
