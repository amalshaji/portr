package utils

import "github.com/mattn/go-sqlite3"

func IsSqliteUniqueConstraintError(err error) bool {
	return err.(sqlite3.Error).ExtendedCode == sqlite3.ErrConstraintUnique
}
