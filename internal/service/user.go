package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	rxEmail   = regexp.MustCompile("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$")
	rxUsename = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_-]{0,17}$")

	// used if user not found in database
	ErrUserNotFound    = errors.New("User Not Found")
	ErrInvalidEmail    = errors.New("This email not invalid")
	ErrInvalidUsername = errors.New("Username is invalid")
	ErrEmailTaken      = errors.New("email taken")
	ErrUsernameTaken   = errors.New("username taken")
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// this function insert user in DB
func (s *Service) CreateUser(ctx context.Context, email, username string) error {

	email = strings.TrimSpace(email)
	if !rxEmail.MatchString(email) {
		return ErrInvalidEmail
	}
	username = strings.TrimSpace(username)
	if !rxUsename.MatchString(username) {
		return ErrInvalidUsername
	}

	query := "INSERT INTO users (email, username) VALUES ($1, $2)"
	_, err := s.db.ExecContext(ctx, query, email, username)

	unique := isUniqueViolation(err)
	if unique && strings.Contains(err.Error(), "email") {
		return ErrEmailTaken
	}
	if unique && strings.Contains(err.Error(), "username") {
		return ErrUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("could not inser user in DB %v", err)
	}
	return nil

}
