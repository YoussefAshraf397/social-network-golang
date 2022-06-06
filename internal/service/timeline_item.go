package service

type TimeLineItem struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"-"`
	PostID int64 `json:"-"`
	Post   Post  `json:"post"`
}
