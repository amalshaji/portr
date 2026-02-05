//go:build nosql

package db

import "github.com/amalshaji/portr/internal/client/config"

// Db is a no-op database implementation used for "nosql" builds.
// It does not read or write anything to disk.
type Db struct{}

func New(_ *config.Config) *Db {
	return &Db{}
}

func (d *Db) LogRequest(_ *RequestLog) error {
	return nil
}
