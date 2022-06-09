package handler

import (
	"github.com/matryer/way"
	"net/http"
	"social-network/internal/service"
)

type handler struct {
	*service.Service
}

func New(s *service.Service) http.Handler {
	h := &handler{s}

	api := way.NewRouter()
	api.HandleFunc("POST", "/login", h.login)
	api.HandleFunc("GET", "/auth_user", h.authUser)
	api.HandleFunc("POST", "/users", h.createUser)
	api.HandleFunc("PUT", "/auth_user/avatar", h.updateavatar)

	api.HandleFunc("GET", "/users/:username", h.user)
	api.HandleFunc("GET", "/users", h.users)
	api.HandleFunc("POST", "/users/:username/toggle_follow", h.toggleFollow)
	api.HandleFunc("GET", "/users/:username/get_followers", h.followers)
	api.HandleFunc("GET", "/users/:username/get_followees", h.followees)
	api.HandleFunc("GET", "users/:username/posts", h.posts)

	api.HandleFunc("POST", "/posts", h.createpost)
	api.HandleFunc("POST", "/posts/:postID/toggle_like", h.togglePostLike)
	api.HandleFunc("POST", "/posts/:postID/toggle_subscription", h.togglePostSubscriptions)

	api.HandleFunc("GET", "/posts/:postID", h.post)

	api.HandleFunc("GET", "/timeline", h.timeline)

	api.HandleFunc("GET", "/posts/:postID/comments", h.comments)
	api.HandleFunc("POST", "/posts/:postID/comments", h.createComment)
	api.HandleFunc("POST", "/comments/:commentID/toggle_like", h.toggleCommentLike)

	api.HandleFunc("GET", "/notifications", h.notifications)
	api.HandleFunc("POST", "/notifications/:notificationID/mark_as_read", h.markNotificationAsRead)
	api.HandleFunc("POST", "/mark_notifications_as_read", h.markAllNotificationAsRead)

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", h.withAuth(api)))
	return r
}
