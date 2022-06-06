package service

// service used for implement the login of function

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	TokenLifeSpan     = time.Hour * 24 * 14
	KeyAuthUserId key = "auth_user_id"
)

type key string

//Login output response
type LoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	AuthUser  User      `json:"authUser"`
}

var (
	ErrUnauthenticated = errors.New("unauthenticated")
)

// Auth user id from token
func (s *Service) AuthUserId(token string) (int64, error) {
	str, err := s.codec.DecodeToString(token)
	if err != nil {
		return 0, fmt.Errorf("could not decode token: %v", err)
	}
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse userId from token: %v", err)
	}
	return i, nil
}

func (s *Service) Login(ctx context.Context, email string) (LoginOutput, error) {
	var out LoginOutput

	email = strings.TrimSpace(email)
	if !rxEmail.MatchString(email) {
		return out, ErrInvalidEmail
	}

	var avatar sql.NullString
	query := "SELECT id, username, avatar  FROM users WHERE email = $1"
	err := s.db.QueryRowContext(ctx, query, email).Scan(&out.AuthUser.ID, &out.AuthUser.Username, &avatar)

	if err == sql.ErrNoRows {
		return out, ErrUserNotFound
	}

	if err != nil {
		return out, fmt.Errorf("could not query select user %v", err)
	}

	if avatar.Valid {
		avatarURL := s.origin + "img/avatars/" + avatar.String
		out.AuthUser.AvatarURL = &avatarURL
	}

	//cretae token
	out.Token, err = s.codec.EncodeToString(strconv.FormatInt(out.AuthUser.ID, 10))
	if err != nil {
		return out, fmt.Errorf("could not create token %v", err)
	}

	out.ExpiresAt = time.Now().Add(TokenLifeSpan)

	return out, nil
}

//AuthUser from context
func (s *Service) AuthUser(ctx context.Context) (User, error) {
	var u User

	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return u, ErrUnauthenticated
	}

	return s.UserByID(ctx, uid)
}
