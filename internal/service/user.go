package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	gonanoid "github.com/matoous/go-nanoid"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	MaxAvatarBytes = 5 << 20 //5MB
)

var (
	rxEmail   = regexp.MustCompile("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$")
	rxUsename = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_-]{0,17}$")
	avatarDir = path.Join("web", "static", "img", "avatars")

	// used if user not found in database
	ErrUserNotFound          = errors.New("User Not Found")
	ErrInvalidEmail          = errors.New("This email not invalid")
	ErrInvalidUsername       = errors.New("Username is invalid")
	ErrEmailTaken            = errors.New("email taken")
	ErrUsernameTaken         = errors.New("username taken")
	ErrForbbiddenFollow      = errors.New("Cannot follow yourself")
	ErrUnsupportAvatarFormat = errors.New("only png and jpeg allowed as avatar")
)

type User struct {
	ID        int64   `json:"id , omitempty"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatarURL"`
}

type UserProfile struct {
	User
	Email          string `json:"email,omitempty"`
	FollowersCount int    `json:"followersCount"`
	FolloweesCount int    `json:"followeesCount"`
	Me             bool   `json:"me"`
	Following      bool   `json:"following"`
	Followeed      bool   `json:"followed"`
}

type ToggleFollowOutput struct {
	Following      bool `json:"following"`
	FollowersCount int  `json:"followersCount"`
}

//CreateUser this function insert user in DB
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
		go s.notifyFollow(followerID, followeeID)

	}
	return out, nil
}

//User select user on user by username in database
func (s *Service) User(ctx context.Context, username string) (UserProfile, error) {

	var u UserProfile

	username = strings.TrimSpace(username)
	if !rxUsename.MatchString(username) {
		return u, ErrInvalidUsername
	}

	uid, auth := ctx.Value(KeyAuthUserId).(int64)

	args := []interface{}{username}
	des := []interface{}{&u.ID, &u.Email, &u.FollowersCount, &u.FolloweesCount}
	query := "SELECT id, email, followers_count, followees_count"
	if auth {
		query += ", " +
			"followers.follower_id IS NOT NULL AS following, " +
			"followees.followee_id IS NOT NULL AS followed"
		des = append(des, &u.Following, &u.Followeed)
	}
	query += " FROM users "
	if auth {
		query += "LEFT JOIN follows AS followers ON followers.follower_id = $2 AND followers.followee_id = users.id " +
			"LEFT JOIN follows AS followees ON followees.follower_id = users.id AND followees.followee_id = $2 "
		args = append(args, uid)
	}
	query += " WHERE username = $1"
	err := s.db.QueryRowContext(ctx, query, args...).Scan(des...)
	if err == sql.ErrNoRows {
		return u, ErrUserNotFound
	}
	if err != nil {
		return u, fmt.Errorf("could not select that user : %v", err)
	}

	u.Username = username
	u.Me = auth && u.ID == uid
	if !u.Me {
		u.ID = 0
		u.Email = ""
	}
	return u, nil
}

//Users list all users with pagination and can search by username
func (s *Service) Users(ctx context.Context, search string, first int, after string) ([]UserProfile, error) {

	search = strings.TrimSpace(search)
	first = normalizePageSize(first)
	after = strings.TrimSpace(after)

	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	query, args, err := buildQuery(`
				SELECT id, email, username, avatar , followers_count, followees_count
				{{if .auth}} 
				, followers.follower_id IS NOT NULL AS following
				, followees.followee_id IS NOT NULL AS followeed
				{{end}}
				FROM users 
				{{if .auth}}
				LEFT JOIN follows AS followers
					ON followers.follower_id = @uid AND followers.followee_id = users.id
				LEFT JOIN follows AS followees 
					ON followees.follower_id = users.id AND followees.followee_id = @uid
				{{end}}
				{{if or .search .after}}WHERE{{end}}
				{{if .search}}username ILIKE '%' || @search || '%'{{end}}
				{{if and .search .after}}AND{{end}}
				{{if .after}}username > @after {{end}}
				ORDER BY username ASC
				LIMIT @first`, map[string]interface{}{
		"auth":   auth,
		"uid":    uid,
		"search": search,
		"first":  first,
		"after":  after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build user sql query: %v", err)
	}
	log.Printf("users query %s\nargs: %v\n", query, args)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select users: %v", err)
	}
	defer rows.Close()
	uu := make([]UserProfile, 0, first)
	for rows.Next() {
		var u UserProfile
		var avatar sql.NullString
		dest := []interface{}{
			&u.ID, &u.Email,
			&u.Username,
			&avatar,
			&u.FollowersCount,
			&u.FolloweesCount,
		}
		if auth {
			dest = append(dest, &u.Following, &u.Followeed)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan user: %w", err)
		}
		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = 0
			u.Email = ""
		}
		if avatar.Valid {
			avatarURL := s.origin + "img/avatars/" + avatar.String
			u.AvatarURL = &avatarURL
		}
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate user rows: %w", err)
	}
	return uu, nil
}

//
func (s *Service) UserByID(ctx context.Context, id int64) (User, error) {
	var u User
	var avatar sql.NullString
	query := "SELECT username , avatar FROM users WHERE id = $1"
	err := s.db.QueryRowContext(ctx, query, id).Scan(&u.Username, &avatar)
	if err == sql.ErrNoRows {
		return u, ErrUserNotFound
	}
	if err != nil {
		return u, fmt.Errorf("could not query select auth user: %v", err)
	}

	u.ID = id
	if avatar.Valid {
		avatarURL := s.origin + "img/avatars/" + avatar.String
		u.AvatarURL = &avatarURL
	}
	return u, nil
}

//Followers list all followers with pagination and can search by username
func (s *Service) Followers(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {

	username = strings.TrimSpace(username)
	if !rxUsename.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	first = normalizePageSize(first)
	after = strings.TrimSpace(after)

	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	query, args, err := buildQuery(`
		SELECT id
		, avatar
		, email
		, username
		, followers_count
		, followees_count
		{{ if .auth }}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{ end }}
				FROM follows
				INNER JOIN users ON follows.follower_id = users.id
		{{ if .auth }}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{ end }}
		WHERE follows.followee_id = ( SELECT id FROM users WHERE username = @username )
		{{ if .after }}AND username > @after{{ end }}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
		"first":    first,
		"after":    after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build followers sql query: %v", err)
	}
	log.Printf("users query %s\nargs: %v\n", query, args)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select followers: %v", err)
	}
	defer rows.Close()
	uu := make([]UserProfile, 0, first)
	for rows.Next() {
		var avatar sql.NullString
		var u UserProfile
		dest := []interface{}{
			&u.ID, &u.Email,
			&avatar,
			&u.Username,
			&u.FollowersCount,
			&u.FolloweesCount,
		}
		if auth {
			dest = append(dest, &u.Following, &u.Followeed)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan followers: %w", err)
		}
		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = 0
			u.Email = ""
		}
		if avatar.Valid {
			avatarURL := s.origin + "img/avatars/" + avatar.String
			u.AvatarURL = &avatarURL
		}

		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate followers rows: %w", err)
	}

	return uu, nil
}

//Followees list all followees with pagination and can search by username
func (s *Service) Followees(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {

	username = strings.TrimSpace(username)
	if !rxUsename.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	first = normalizePageSize(first)
	after = strings.TrimSpace(after)

	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	query, args, err := buildQuery(`
				SELECT id, avatar , email, username, followers_count, followees_count
				{{if .auth}} 
				, followers.follower_id IS NOT NULL AS following
				, followees.followee_id IS NOT NULL AS followeed
				{{end}}
				FROM follows
					INNER JOIN USERS ON follows.followee_id = users.id
				{{if .auth}}
				LEFT JOIN follows AS followers
					ON followers.follower_id = @uid AND followers.followee_id = users.id
				LEFT JOIN follows AS followees 
					ON followees.follower_id = users.id AND followees.followee_id = @uid
				{{end}}
				WHERE follows.follower_id = ( SELECT id FROM users WHERE username = @username)
				{{if  .after}} AND username > @after {{end}}
				ORDER BY username ASC
				LIMIT @first`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
		"first":    first,
		"after":    after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build followees sql query: %v", err)
	}
	log.Printf("users query %s\nargs: %v\n", query, args)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select followees: %v", err)
	}
	defer rows.Close()
	uu := make([]UserProfile, 0, first)
	for rows.Next() {
		var avatar sql.NullString
		var u UserProfile
		dest := []interface{}{
			&u.ID, &u.Email,
			&avatar,
			&u.Username,
			&u.FollowersCount,
			&u.FolloweesCount,
		}
		if auth {
			dest = append(dest, &u.Following, &u.Followeed)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan followees: %w", err)
		}
		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = 0
			u.Email = ""
		}
		if avatar.Valid {
			avatarURL := s.origin + "img/avatars/" + avatar.String
			u.AvatarURL = &avatarURL
		}
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate followees rows: %w", err)
	}
	return uu, nil
}

//UpdateAvatar update avatar of auth user
func (s *Service) UpdateAvatar(ctx context.Context, r io.Reader) (string, error) {
	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	if !auth {
		return "", ErrUnauthenticated
	}
	r = io.LimitReader(r, MaxAvatarBytes)
	img, format, err := image.Decode(r)
	if err != nil {
		return "", fmt.Errorf("Could not read avatar: %v", err)
	}
	if format != "png" && format != "jpeg" {
		return "", ErrUnsupportAvatarFormat
	}

	avatar, err := gonanoid.Nanoid()
	if err != nil {
		return "", fmt.Errorf("Could not generate filename: %v", err)
	}

	if format == "png" {
		avatar += ".png"
	} else {
		avatar += ".jpg"
	}

	avatarPath := path.Join(avatarDir, avatar)
	f, err := os.Create(avatarPath)
	if err != nil {
		return "", fmt.Errorf("Could not create avatar file: %v", err)
	}
	defer f.Close()

	img = imaging.Fill(img, 400, 400, imaging.Center, imaging.CatmullRom)
	if format == "png" {
		err = png.Encode(f, img)
	} else {
		err = jpeg.Encode(f, img, nil)
	}
	if err != nil {
		return "", fmt.Errorf("Could not write avatar to desk: %v", err)
	}

	var oldAvatar sql.NullString
	query := `UPDATE users SET avatar = $1 WHERE id = $2
				RETURNING ( SELECT avatar FROM users WHERE id = $2 ) AS old_avatar`

	if err = s.db.QueryRowContext(ctx, query, avatar, uid).Scan(&oldAvatar); err != nil {
		defer os.Remove(avatarPath)
		return "", fmt.Errorf("Could not update avatar: %v", err)
	}
	if oldAvatar.Valid {
		defer os.Remove(path.Join(avatarDir, oldAvatar.String))

	}

	return s.origin + "/img/avatars" + avatar, nil

}
