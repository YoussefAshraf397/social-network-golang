package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

type TimeLineItem struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"-"`
	PostID int64 `json:"-"`
	Post   Post  `json:"post"`
}

type timelineItemClient struct {
	timeline chan TimeLineItem
	userID   int64
}

func (s *Service) Timeline(ctx context.Context, last int, before int64) ([]TimeLineItem, error) {
	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}
	last = normalizePageSize(last)

	q := `	SELECT timeline.id, posts.id, content, spoiler_of, nsfw, likes_count, comments_count, created_at
				, posts.user_id = @uid AS mine
				, likes.user_id IS NOT NULL AS liked
				, subscriptions.user_id IS NOT NULL AS subscribed
				, users.username, users.avatar
				
				FROM timeline
				INNER JOIN posts ON timeline.post_id = posts.id
				INNER JOIN users ON timeline.user_id = users.id
				LEFT JOIN post_likes AS likes
					ON likes.user_id = @uid AND likes.post_id = posts.id
				LEFT JOIN post_subscriptions AS subscriptions
					ON subscriptions.user_id = @uid AND subscriptions.post_id = posts.id
				WHERE timeline.user_id = @uid
				{{if .before}}AND timeline.id < @before{{end}}
				ORDER BY created_at DESC
				LIMIT @last`

	query, args, err := buildQuery(q, map[string]interface{}{
		"uid":    uid,
		"last":   last,
		"before": before,
	})

	if err != nil {
		return nil, fmt.Errorf("could not build timeline sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select timeline:  %v", err)
	}
	defer rows.Close()

	tt := make([]TimeLineItem, 0, last)
	for rows.Next() {
		var t TimeLineItem
		var u User
		var avatar sql.NullString
		dest := []interface{}{
			&t.ID,
			&t.Post.ID,
			&t.Post.Content,
			&t.Post.SpoilerOf,
			&t.Post.NSFW,
			&t.Post.LikesCount,
			&t.Post.CommentsCount,
			&t.Post.CreatedAt,
			&t.Post.Mine,
			&t.Post.Liked,
			&t.Post.Subscribed,
			&u.Username,
			&avatar,
		}

		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan timeline: %v", err)
		}

		if avatar.Valid {
			avatarURL := s.origin + "img/avatars/" + avatar.String
			u.AvatarURL = &avatarURL
		}

		t.Post.User = &u

		go s.broadcastTimelineItem(t)
		//tt = append(tt, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate timeline rows: %v", err)
	}

	return tt, nil
}

//SubscribedToTimeline to receive timeline item in realtime
func (s *Service) SubscribedToTimeline(ctx context.Context) (chan TimeLineItem, error) {
	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}

	tt := make(chan TimeLineItem)
	c := &timelineItemClient{timeline: tt, userID: uid}
	s.timelineItemClients.Store(c, struct{}{})

	go func() {
		<-ctx.Done()
		s.timelineItemClients.Delete(c)
		close(tt)

	}()

	return tt, nil
}

func (s *Service) broadcastTimelineItem(ti TimeLineItem) {
	s.timelineItemClients.Range(func(key, value interface{}) bool {
		client := key.(*timelineItemClient)
		if client.userID == ti.UserID {
			log.Printf(" test timeline", ti)

			client.timeline <- ti
		}
		return true
	})
}
