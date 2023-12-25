package db

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/amalshaji/localport/internal/server/admin/migrations"
	"github.com/amalshaji/localport/internal/server/config"
	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/lopezator/migrator"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

type Db struct {
	Conn    *sql.DB
	Queries *db.Queries
	config  *config.DatabaseConfig
}

func New(config *config.DatabaseConfig) *Db {
	return &Db{
		config: config,
	}
}

var (
	DefaultSmtpEnabled            = false
	DefaultAddMemberEmailSubject  = utils.Trim("You've been added to team {{teamName}} on LocalPort!")
	DefaultAddMemberEmailTemplate = utils.Trim(`Hello {{email}}
	
You've been added to team "{{teamName}}" on LocalPort.

Get started by signing in with your github account at {{appUrl}}`)
)

func (d *Db) Connect() {
	var err error

	d.Conn, err = sql.Open("libsql", d.config.Url)
	if err != nil {
		log.Fatal(err)
	}

	d.Queries = db.New(d.Conn)
}

func (d *Db) populateDefaultSettings(ctx context.Context) {
	_, err := d.Queries.GetGlobalSettings(ctx)

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

func (d *Db) Migrate() {
	if d.Conn == nil {
		log.Fatal("db connection not initialized")
	}

	allMigrations := []any{}

	for name, migration := range migrations.MigrationMap {
		allMigrations = append(allMigrations, &migrator.Migration{
			Name: name,
			Func: func(tx *sql.Tx) error {
				_, err := migration(tx)
				return err
			},
		})

		_m, err := migrator.New(migrator.Migrations(allMigrations...))
		if err != nil {
			log.Fatalf("error creating migrator: %v", err)
		}

		if err := _m.Migrate(d.Conn); err != nil {
			log.Fatalf("error running migrations: %v", err)
		}
	}

	ctx := context.Background()
	d.populateDefaultSettings(ctx)
}
