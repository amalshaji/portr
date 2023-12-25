package utils

import (
	"modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

func IsSqliteUniqueConstraintError(err error) bool {
	return err.(*sqlite.Error).Code() == sqlitelib.SQLITE_CONSTRAINT_UNIQUE
}
