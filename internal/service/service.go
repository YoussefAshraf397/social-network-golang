package service

import (
	"database/sql"
	"github.com/hako/branca"
)

type Service struct {
	db     *sql.DB
	codec  *branca.Branca
	origin string
}

// New service implementation
func New(db *sql.DB, codec *branca.Branca, origin string) *Service {
	return &Service{
		db:     db,
		codec:  codec,
		origin: origin,
	}
}
