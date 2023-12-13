package db

import (
	"context"
	"database/sql"
	"errors"
	"log"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/utils"
	_ "github.com/mattn/go-sqlite3"
)

type Db struct {
	Conn    *sql.DB
	Queries *db.Queries
}

func New() *Db {
	return &Db{}
}

var (
	DefaultUserInviteEmailSubject  = utils.Trim("Invitation to join team {{teamName}} on LocalPort")
	DefaultUserInviteEmailTemplate = utils.Trim(`Hi {{email}}, you have been invited to join team {{teamName}} on LocalPort. You can now signup using your GitHub account.

{{appUrl}}`)
)

func (d *Db) Connect() {
	var err error

	d.Conn, err = sql.Open("sqlite3", "./data/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	d.Queries = db.New(d.Conn)
	_, err = d.Queries.GetGlobalSettings(ctx)
	// Populate/update default settings

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = d.Queries.CreateGlobalSettings(ctx, db.CreateGlobalSettingsParams{
				UserInviteEmailTemplate: DefaultUserInviteEmailTemplate,
				UserInviteEmailSubject:  DefaultUserInviteEmailSubject,
			})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}
}
