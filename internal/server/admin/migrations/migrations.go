package migrations

import (
	"database/sql"
)

var MigrationMap = map[string]func(tx *sql.Tx) (sql.Result, error){
	"create_all_tables": V_001,
}
