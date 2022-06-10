package service

import (
	"context"
	"database/sql"
	"github.com/hako/branca"
	"net"
	"net/smtp"
	"net/url"
	"strconv"
	"sync"
)

type Service struct {
	db                  *sql.DB
	codec               *branca.Branca
	origin              string
	noReply             string
	smtpAddr            string
	smtpAuth            smtp.Auth
	timelineItemClients sync.Map
	commentClients      sync.Map
	notificationClients sync.Map
}

//config to create a new service
type Config struct {
	DB           *sql.DB
	SecretKey    string
	Origin       string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
}

// New service implementation
func New(cfg Config) *Service {
	codec := branca.NewBranca(cfg.SecretKey)
	codec.SetTTL(uint32(tokenLifeSpan.Seconds()))

	originURL, _ := url.Parse(cfg.Origin)

	s := &Service{
		db:       cfg.DB,
		codec:    codec,
		origin:   cfg.Origin,
		noReply:  "noreply@" + originURL.Hostname(),
		smtpAddr: net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort)),
		smtpAuth: smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost),
	}

	go s.deleteExpiredVerificationCodesCronJob(context.Background())
	return s

}
