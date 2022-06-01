package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	rxEmail   = regexp.MustCompile("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$")
	rxUsename = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_-]{0,17}$")

	// used if user not found in database
	ErrUserNotFound     = errors.New("User Not Found")
	ErrInvalidEmail     = errors.New("This email not invalid")
	ErrInvalidUsername  = errors.New("Username is invalid")
	ErrEmailTaken       = errors.New("email taken")
	ErrUsernameTaken    = errors.New("username taken")
	ErrForbbiddenFollow = errors.New("Cannot follow yourself")
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type ToggleFollowOutput struct {
	Following      bool `json:"following"`
	FollowersCount int  `json:"followersCount"`
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

//ToggleFollow this function recive the name of you want follow
func (s *Service) ToggleFollow(ctx context.Context, username string) (ToggleFollowOutput, error) {
	var out ToggleFollowOutput
	followerID, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return out, ErrUnauthenticated
	}

	username = strings.TrimSpace(username)
	if !rxUsename.MatchString(username) {
		return out, ErrInvalidUsername
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("could not begin tx: %v", err)
	}

	defer tx.Rollback()
	var followeeID int64
	query := "SELECT id FROM users WHERE username = $1"
	err = tx.QueryRowContext(ctx, query, username).Scan(&followeeID)
	if err == sql.ErrNoRows {
		return out, ErrUserNotFound
	}

	if err != nil {
		return out, fmt.Errorf("could not query select user id from followee username: %v", err)
	}

	if followerID == followeeID {
		return out, ErrForbbiddenFollow
	}

	query = "SELECT EXISTS (SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2)"
	if err := tx.QueryRowContext(ctx, query, followerID, followeeID).Scan(&out.Following); err != nil {
		return out, fmt.Errorf("could not query select existance of follow: %v", err)
	}

	if out.Following {
		query = "DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2"
		if _, err = tx.ExecContext(ctx, query, followerID, followeeID); err != nil {
			return out, fmt.Errorf("could not delete follow: %d", err)
		}
		query = "UPDATE users set followees_count = followees_count - 1 WHERE id = $1"
		if _, err = tx.ExecContext(ctx, query, followerID); err != nil {
			return out, fmt.Errorf("Could not update followe folowee_count: %v", err)
		}

		query = "UPDATE users set followers_count = followers_count - 1 WHERE id = $1 RETURNING followers_count"
		if err = tx.QueryRowContext(ctx, query, followeeID).Scan(&out.FollowersCount); err != nil {
			return out, fmt.Errorf(" could not update followee followers conut: %v", err)
		}

	} else {
		query = "INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2)"
		_, err = tx.ExecContext(ctx, query, followerID, followeeID)
		if err != nil {
			return out, fmt.Errorf("could not insert follow: %w", err)
		}

		query = "UPDATE users SET followees_count = followees_count + 1 WHERE id = $1"
		if _, err = tx.ExecContext(ctx, query, followerID); err != nil {
			return out, fmt.Errorf("could not increment followees count: %w", err)
		}
		query = "UPDATE users SET followers_count = followers_count + 1 WHERE id = $1 RETURNING followers_count"
		row := tx.QueryRowContext(ctx, query, followeeID)
		err = row.Scan(&out.FollowersCount)
		if err != nil {
			return out, fmt.Errorf("could not increment followers count: %w", err)
		}

	}

	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("could not commit toggle follow: %d", err)
	}

	out.Following = !out.Following

	if out.Following {
		//TODO: notify user followee

	}
	return out, nil
}
