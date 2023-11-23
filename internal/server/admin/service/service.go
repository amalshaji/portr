package service

import "github.com/amalshaji/localport/internal/server/db"

type Service struct {
	db *db.Db
}

func New(db *db.Db) *Service {
	return &Service{db: db}
}
