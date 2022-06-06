package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/sanity-io/litter"
	"log"
	"strings"
	"time"
)

var (
	ErrInvalidContnt  = errors.New("invalid content")
	ErrInvalidSpoiler = errors.New("invalid spoiler")
)

type Post struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	Content   string    `json:"content"`
	SpoilerOf *string   `json:"spoilerOf"`
	NSFW      bool      `json:"nsfw"`
	CreatedAt time.Time `json:"createdAt"`
	User      *User     `json:"user, omitempty"`
	Mine      bool      `json:"mine"`
}

//CreatePost user publish post to timeline
func (s *Service) CreatePost(ctx context.Context, content string, spoilerOf *string, nsfw bool) (TimeLineItem, error) {
	var ti TimeLineItem
	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return ti, ErrUnauthenticated
	}
	content = strings.TrimSpace(content)
	if content == "" || len(content) > 400 {
		return ti, ErrInvalidContnt
	}

	if spoilerOf != nil {
		*spoilerOf = strings.TrimSpace(*spoilerOf)
		if *spoilerOf == "" || len([]rune(*spoilerOf)) > 64 {
			return ti, ErrInvalidSpoiler
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ti, fmt.Errorf("Could not begin tx: %v", err)
	}
	defer tx.Rollback()
	query := "INSERT INTO posts (user_id , content , spoiler_of , nsfw) VALUES ($1, $2, $3, $4)" +
		"RETURNING id, created_at"
	if err = tx.QueryRowContext(ctx, query, uid, content, spoilerOf, nsfw).Scan(&ti.Post.ID, &ti.Post.CreatedAt); err != nil {
		return ti, fmt.Errorf("Could not insert post to database tx: %v", err)
	}

	ti.Post.UserID = uid
	ti.Post.Content = content
	ti.Post.SpoilerOf = spoilerOf
	ti.Post.NSFW = nsfw
	ti.Post.Mine = true

	query = "INSERT INTO timeline (user_id , post_id) VALUES ($1, $2) RETURNING id"
	if err = tx.QueryRowContext(ctx, query, uid, ti.Post.ID).Scan(&ti.ID); err != nil {
		return ti, fmt.Errorf("Could not insert timeline to database tx: %v", err)
	}

	ti.UserID = uid
	ti.PostID = ti.Post.ID

	if err = tx.Commit(); err != nil {
		return ti, fmt.Errorf("could not commit to create post: %v", err)
	}

	go func(p Post) {
		u, err := s.UserByID(context.Background(), p.UserID)
		if err != nil {
			log.Printf("could not get post user %v\n", err)
			return
		}
		p.User = &u
		p.Mine = false

		tt, err := s.fanoutPost(p)
		if err != nil {
			log.Printf("could not fanout post: %v\n", err)
		}
		for _, ti = range tt {
			log.Println(litter.Sdump(ti))

		}

	}(ti.Post)

	return ti, nil
}

func (s *Service) fanoutPost(p Post) ([]TimeLineItem, error) {
	query := "INSERT INTO timeline (user_id, post_id)" +
		"SELECT follower_id, $1 FROM follows WHERE followee_id = $2 " +
		"RETURNING id, user_id"
	rows, err := s.db.Query(query, p.ID, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("could not insert to timeline: %v", err)
	}
	defer rows.Close()

	tt := []TimeLineItem{}
	for rows.Next() {
		var ti TimeLineItem
		if err = rows.Scan(&ti.ID, &ti.UserID); err != nil {
			return nil, fmt.Errorf("could not scan timeline item: %v", err)
		}
		ti.PostID = p.ID
		ti.Post = p

		tt = append(tt, ti)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate timeline records: %v", err)
	}

	return tt, nil

}
