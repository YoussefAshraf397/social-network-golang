package service

import (
	"database/sql"
	"github.com/hako/branca"
	"sync"
)

type Service struct {
	db                  *sql.DB
	codec               *branca.Branca
	origin              string
	timelineItemClients sync.Map
	commentClients      sync.Map
	notificationClients sync.Map
}

// New service implementation
func New(db *sql.DB, codec *branca.Branca, origin string) *Service {
	return &Service{
		db:     db,
		codec:  codec,
		origin: origin,
	}
}
