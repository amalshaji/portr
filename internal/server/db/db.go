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
	DefaultSignupRequiresInvite           = true
	DefaultAllowRandomUserSignup          = false
	DefaultRandomUserSignupAllowedDomains = ""
	DefaultUserInviteEmailTemplate        = utils.Trim(`Hi {{email}}, you have been invited to join LocalPort. Click the link below to get started.

<a href="{{inviteUrl}}">Click here to create your account</a>`)
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
		&User{},
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
		settings.AllowRandomUserSignup = DefaultAllowRandomUserSignup
		settings.SignupRequiresInvite = DefaultSignupRequiresInvite
		settings.RandomUserSignupAllowedDomains = DefaultRandomUserSignupAllowedDomains
		settings.UserInviteEmailTemplate = DefaultUserInviteEmailTemplate

		result := d.Conn.Save(&settings)
		if result.Error != nil {
			log.Fatal(result.Error)
		}
	}

}
