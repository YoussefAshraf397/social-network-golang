package service

// service used for implement the login of function

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	verificationCodeLifeSpan = time.Minute * 15
	tokenLifeSpan            = time.Hour * 24 * 14

	KeyAuthUserId key = "auth_user_id"
)

var magicLinkTempl *template.Template

type key string

//Login output response
type LoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	AuthUser  User      `json:"authUser"`
}

type TokenOutput struct {
	Token     string
	ExpiresAt time.Time
}

var rxUUID = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

var (
	ErrUnauthenticated          = errors.New("unauthenticated")
	ErrInvalidRedirectURI       = errors.New("invalid redirect uri")
	ErrInvalidVerificationCode  = errors.New("invalid verification code")
	ErrVerificationCodeNotFound = errors.New("verification code not found")
	ErrVerificationCodeExpired  = errors.New("verification code is expired")
)

//SendMagicLink to can login without password
func (s *Service) SendMagicLink(ctx context.Context, email, redirectURI string) error {
	email = strings.TrimSpace(email)
	if !rxEmail.MatchString(email) {
		return ErrInvalidEmail
	}

	uri, err := url.ParseRequestURI(redirectURI)
	if err != nil {
		return ErrInvalidRedirectURI
	}

	query := `INSERT INTO verification_codes (user_id) VALUES (
(SELECT id FROM users WHERE email = $1)
) RETURNING id`
	var verificationCode string
	err = s.db.QueryRowContext(ctx, query, email).Scan(&verificationCode)
	if isForeignKeyViolation(err) {
		return ErrUserNotFound
	}
	//TODO: check email is exist or not
	if err != nil {
		return fmt.Errorf("could not insert verification code: %v", err)
	}

	var code string
	err = s.db.QueryRowContext(ctx, `
INSERT INTO verification_codes (user_id) VALUES  (
									 ( SELECT id FROM userHERE email = $1)
) RETURNING id`, email).Scan(&code)

	if isForeignKeyViolation(err) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("could not inser verification code: %v", err)
	}

	magicLink, _ := url.Parse(s.origin)
	magicLink.Path = "/api/auth_redirect"

	q := magicLink.Query()
	q.Set("verification_code", verificationCode)
	q.Set("redirect_uri", uri.String())
	magicLink.RawQuery = q.Encode()

	if magicLinkTempl == nil {
		magicLinkTempl, err = template.ParseFiles("web/template/mail/magic-link.html")
		if err != nil {
			return fmt.Errorf("could not parse magic link template: %v", err)
		}
	}

	var mail bytes.Buffer
	if err = magicLinkTempl.Execute(&mail, map[string]interface{}{
		"MagicLink": magicLink,
		"Minutes":   int(verificationCodeLifeSpan.Minutes()),
	}); err != nil {
		return fmt.Errorf("could not execute magic link template: %v", err)
	}

	//s.sender.Send(email, "Magic Link", mail.String())
	if err = s.sendMail(email, "Magic Link", mail.String()); err != nil {
		return fmt.Errorf("could not send magic link: %v", err)
	}

	go s.deleteExpiredVerificationCodesCronJob(code)

	return nil
}

//AuthURI to can be redirect and complete login flow
//token is hash fragment
func (s *Service) AuthURI(ctx context.Context, verificationCode, redirectURI string) (string, error) {
	verificationCode = strings.TrimSpace(verificationCode)
	if !rxUUID.MatchString(verificationCode) {
		return "", ErrInvalidVerificationCode
	}

	uri, err := url.ParseRequestURI(redirectURI)
	if err != nil {
		return "", ErrInvalidRedirectURI
	}

	var uid int64
	var ts time.Time
	query := `DELETE FROM verification_codes WHERE id = $1 RETURNING user_id, created_at`
	err = s.db.QueryRowContext(ctx, query, verificationCode).Scan(&uid, &ts)
	if err == sql.ErrNoRows {
		return "", ErrVerificationCodeNotFound
	}

	if err != nil {
		return "", fmt.Errorf("could not delete verification code: %v", err)
	}

	if ts.Add(verificationCodeLifeSpan).Before(time.Now()) {
		return "", ErrVerificationCodeExpired
	}

	token, err := s.codec.EncodeToString(strconv.FormatInt(uid, 10))
	if err != nil {
		return "", fmt.Errorf("could not create token: %v", err)
	}

	exp, err := time.Now().Add(tokenLifeSpan).MarshalText()
	if err != nil {
		return "", fmt.Errorf("could not marshall token lifspan: %v", err)
	}

	f := url.Values{}
	f.Set("token", token)
	f.Set("expires_at", string(exp))
	uri.Fragment = f.Encode()

	return uri.String(), nil

}

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

	out.ExpiresAt = time.Now().Add(tokenLifeSpan)

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

func (s *Service) Token(ctx context.Context) (TokenOutput, error) {
	var out TokenOutput

	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return out, ErrUnauthenticated
	}

	var err error
	out.Token, err = s.codec.EncodeToString(strconv.FormatInt(uid, 10))
	if err != nil {
		return out, fmt.Errorf("could not create token: %v", err)
	}

	out.ExpiresAt = time.Now().Add(tokenLifeSpan)
	return out, nil

}

func (s *Service) deleteExpiredVerificationCodesCronJob(code string) {
	<-time.After(verificationCodeLifeSpan)
	if _, err := s.db.Exec(`DELETE FROM verification_codes WHERE id = $1`, code); err != nil {
		log.Printf("could not delete expired verfication code: %v\n", err)
	}
}
