package service

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
	"time"
)

//EX1:youssef mentioned you in post
//EX2:youssef followed you
//EX3:youssef commented on post

//Notification model.
type Notification struct {
	ID       int64     `json:"id"`
	UserID   int64     `json:"-"`
	Actors   []string  `json:"actors"`
	Type     string    `json:"type"`
	PostID   *int64    `json:"postID,omitempty"`
	Read     bool      `json:"read"`
	IssuedAt time.Time `json:"issuedAt"`
}

type notificationClient struct {
	notifications chan Notification
	userID        int64
}

//Notifications List notifications of the auth user
func (s *Service) Notifications(ctx context.Context, last int, before int64) ([]Notification, error) {

	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}
	last = normalizePageSize(last)

	q := `SELECT id, actors, type, post_id, read, issued_at
FROM notifications 
WHERE user_id = @uid
{{if .before}}AND id < @before {{end}}
ORDER BY issued_at DESC
LIMIT @last`

	query, args, err := buildQuery(q, map[string]interface{}{
		"uid":    uid,
		"before": before,
		"last":   last,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build list notifications query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select notifications: %v", err)
	}
	defer rows.Close()

	nn := make([]Notification, 0, last)
	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, pq.Array(&n.Actors), &n.Type, &n.PostID, &n.Read, &n.IssuedAt); err != nil {
			return nil, fmt.Errorf("could not scan notifications: %v", err)
		}
		nn = append(nn, n)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate over notifications rows: %v", err)
	}
	return nn, nil
}

//MarkNotificationAsRead set notification of the auth user as read
func (s *Service) MarkNotificationAsRead(ctx context.Context, notificationID int64) error {
	uid, ok := ctx.Value(KeyAuthUserId).(int64)

	if !ok {
		return ErrUnauthenticated
	}

	q := `UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`
	if _, err := s.db.Exec(q, notificationID, uid); err != nil {
		return fmt.Errorf("could not update and mark notification of auth user as read: %v", err)
	}
	return nil

}

//MarkAllNotificationAsRead set all notifications of auth user as read.
func (s *Service) MarkAllNotificationAsRead(ctx context.Context) error {
	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	log.Println(uid)
	if !ok {
		return ErrUnauthenticated
	}

	q := `UPDATE notifications SET read = true WHERE user_id = $1`
	if _, err := s.db.Exec(q, uid); err != nil {
		return fmt.Errorf("could not update and mark notifications as read: %v", err)
	}
	return nil

}

func (s *Service) HasUnreadNotifications(ctx context.Context) (bool, error) {
	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return false, ErrUnauthenticated
	}

	var unread bool
	query := `SELECT EXISTS (
			 	SELECT 1 from notifications WHERE user_id = $1 AND read = false
			)`
	if err := s.db.QueryRowContext(ctx, query, uid).Scan(&unread); err != nil {
		return false, fmt.Errorf("could not query select un read notifications existence: %v", err)
	}

	return unread, nil
}

func (s *Service) notifyFollow(followerID, followeeID int64) {
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("could not begin tx: %v", err)
		return
	}
	defer tx.Rollback()

	var actor string
	query := "SELECT username FROM users WHERE id = $1"
	if err = tx.QueryRow(query, followerID).Scan(&actor); err != nil {
		log.Printf("could not query select follow notification actor: %v\n", err)
		return
	}

	var notified bool
	query = `SELECT EXISTS(
    SELECT 1 FROM notifications
    WHERE user_id = $1
    	AND $2:::VARCHAR = ANY(actors)
    	AND type = 'follow'
		)`

	if err = tx.QueryRow(query, followeeID, actor).Scan(&notified); err != nil {
		log.Printf("could not query select follow notification existence: %v\n", err)
		return
	}
	if notified {
		return
	}

	var nid int64
	query = "SELECT id FROM notifications WHERE user_id = $1 AND type = 'follow' AND read=false"
	err = tx.QueryRow(query, followeeID).Scan(&nid)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("could not query select unread follow notification existence: %v\n", err)
		return
	}

	var n Notification
	if err == sql.ErrNoRows {
		actors := []string{actor}
		query = `
    		INSERT INTO notifications (user_id, actors, type) VALUES($1, $2, 'follow') 
    		RETURNING id, issued_at `
		if err = tx.QueryRow(query, followeeID, pq.Array(actors)).Scan(&n.ID, &n.IssuedAt); err != nil {
			log.Printf("could not insert follow notifications: %v\n", err)
			return
		}
		n.Actors = actors

	} else {
		query = `
 				UPDATE notifications SET 
					actors = array_prepend($1, notifications.actors),
					issued_at = now()
				WHERE id = $2
				RETURNING actors, issued_at`
		if err = tx.QueryRow(query, actor, nid).Scan(pq.Array(&n.Actors), &n.IssuedAt); err != nil {
			log.Printf("could not update follow notifications: %v\n", err)
			return
		}
		n.ID = nid

	}
	n.UserID = followeeID
	n.Type = "follow"

	if err = tx.Commit(); err != nil {
		log.Printf("could not commit tx of notifications foillow: %v\n", err)
		return
	}

	go s.broadcastNotification(n)

	//TODO: Broadcat follow notification.
}

func (s *Service) notifyComment(c Comment) {

	actor := c.User.Username
	q := `
			INSERT INTO notifications (user_id, actors, type, post_id)
			SELECT user_id, $1, 'comment', $2 FROM post_subscriptions
			WHERE post_subscriptions.user_id != $3
				AND post_subscriptions.post_id = $2
			ON CONFLICT (user_id, type, post_id, read) DO UPDATE SET
				actors = array_prepend($4, array_remove(notifications.actors, $4)) ,
				issued_At = now()
			RETURNING id, user_id, actors, issued_At`
	rows, err := s.db.Query(q, pq.Array([]string{actor}),
		c.PostID, c.UserID, actor)
	if err != nil {
		log.Printf("could not insert comment notifications: %v\n", err)
	}
	defer rows.Close()

	//nn := make([]Notification, 0)
	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.UserID, pq.Array(&n.Actors), &n.IssuedAt); err != nil {
			log.Printf("could not scan comment notification: %v\n", err)
			return
		}
		n.Type = "comment"
		n.PostID = &c.PostID

		go s.broadcastNotification(n)

		//nn = append(nn, n)
	}

	if err = rows.Err(); err != nil {
		log.Printf("could not iterate comment notification: %v\n", err)
		return
	}

	//TODO: Broadcast comment notification
}

func (s *Service) notifyPostMention(p Post) {
	mentions := collectMentions(p.Content)
	log.Printf("mentions: %v+v\n", mentions)

	if len(mentions) == 0 {
		log.Println("no mentions")

		return
	}
	actors := []string{p.User.Username}

	q := `
		INSERT INTO notifications (user_id, actors, type, post_id)
		SELECT users.id , $1 , 'post_mention' , $2 FROM users
		WHERE users.id != $3
			AND username = ANY($4)
		RETURNING id, user_id, issued_at`
	rows, err := s.db.Query(q, pq.Array(actors), p.ID, p.UserID, pq.Array(mentions))
	if err != nil {
		log.Printf("could not insert post mention notification: %v\n", err)
		return
	}
	defer rows.Close()

	//nn := []Notification{}
	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.UserID, &n.IssuedAt); err != nil {
			log.Printf("could not scan comment notification: %v\n", err)
			return
		}
		n.Actors = actors
		n.Type = "post_mention"
		n.PostID = &p.ID
		go s.broadcastNotification(n)

		//nn = append(nn, n)
	}
	if err = rows.Err(); err != nil {
		log.Printf("could not iterate post mention notification: %v\n", err)
		return
	}
	//log.Printf("notifications: %v+v\n", nn)

	//TODO: Broadcast post mention notification.

}

func (s *Service) notifyCommentMention(c Comment) {
	mentions := collectMentions(c.Content)
	log.Printf("mentions: %v+v\n", mentions)

	if len(mentions) == 0 {
		log.Println("no mentions.")
		return
	}
	actor := c.User.Username

	q := `
		INSERT INTO notifications (user_id, actors, type, post_id)
		SELECT users.id , $1 , 'comment_mention' , $2 FROM users
		WHERE users.id != $3
			AND username = ANY($4)
		ON CONFLICT (user_id , type , post_id , read) DO UPDATE SET
			actors = array_prepend($5 , array_remove(notifications.actors , $5)) ,
			issued_at = now()
		RETURNING id, user_id, actors, issued_at`
	rows, err := s.db.Query(q, pq.Array([]string{actor}), c.PostID, c.UserID, pq.Array(mentions), actor)
	if err != nil {
		log.Printf("could not insert comment mention notification: %v\n", err)
		return
	}
	defer rows.Close()

	//nn := []Notification{}
	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.UserID, pq.Array(&n.Actors), &n.IssuedAt); err != nil {
			log.Printf("could not scan comment notification: %v\n", err)
			return
		}
		n.Type = "comment_mention"
		n.PostID = &c.PostID
		go s.broadcastNotification(n)

		//nn = append(nn, n)
	}
	if err = rows.Err(); err != nil {
		log.Printf("could not iterate comment mention notification: %v\n", err)
		return
	}
	//log.Printf("notifications: %v+v\n", nn)
	//TODO: Broadcast comment mention notification.

}

//SubscribedToNotifications to receive notification in realtime
func (s *Service) SubscribedToNotifications(ctx context.Context) (chan Notification, error) {
	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}

	nn := make(chan Notification)
	c := &notificationClient{notifications: nn, userID: uid}
	s.notificationClients.Store(c, struct{}{})

	go func() {
		<-ctx.Done()
		s.notificationClients.Delete(c)
		close(nn)

	}()

	return nn, nil
}

func (s *Service) broadcastNotification(n Notification) {
	s.notificationClients.Range(func(key, value interface{}) bool {
		client := key.(*notificationClient)
		if client.userID == n.UserID {
			client.notifications <- n
		}
		return true
	})
}
