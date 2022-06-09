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
	Read     bool      `json:"read"`
	IssuedAt time.Time `json:"issuedAt"`
}

//Notifications List notifications of the auth user
func (s *Service) Notifications(ctx context.Context, last int, before int64) ([]Notification, error) {

	uid, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}
	last = normalizePageSize(last)

	q := `SELECT id, actors, type, read, issued_at
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
		if err = rows.Scan(&n.ID, pq.Array(&n.Actors), &n.Type, &n.Read, &n.IssuedAt); err != nil {
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
	//TODO: Broadcat follow notification.
}