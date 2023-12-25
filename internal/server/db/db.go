package db

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/amalshaji/localport/internal/server/config"
	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/utils"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

type Db struct {
	Conn    *sql.DB
	Queries *db.Queries
}

func New() *Db {
	return &Db{}
}

var (
	DefaultSmtpEnabled            = false
	DefaultAddMemberEmailSubject  = utils.Trim("You've been added to team {{teamName}} on LocalPort!")
	DefaultAddMemberEmailTemplate = utils.Trim(`Hello {{email}}
	
You've been added to team "{{teamName}}" on LocalPort.

Get started by signing in with your github account at {{appUrl}}`)
)

func (d *Db) Connect(config *config.Config) {
	var err error

	d.Conn, err = sql.Open("libsql", config.DatabaseUrl)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	d.Queries = db.New(d.Conn)
	_, err = d.Queries.GetGlobalSettings(ctx)

	// Populate default settings
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = d.Queries.CreateGlobalSettings(ctx, db.CreateGlobalSettingsParams{
				SmtpEnabled:            DefaultSmtpEnabled,
				AddMemberEmailSubject:  DefaultAddMemberEmailSubject,
				AddMemberEmailTemplate: DefaultAddMemberEmailTemplate,
			})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}
}
