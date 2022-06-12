package handler

import (
	"github.com/matryer/way"
	"mime"
	"net/http"
	"social-network/internal/service"
	"time"
)

type handler struct {
	*service.Service
	ping time.Duration
}

func New(s *service.Service, ping time.Duration, inLocalhost bool) http.Handler {
	h := &handler{Service: s, ping: ping}

	api := way.NewRouter()
	api.HandleFunc("POST", "/send_magic_link", h.sendMagicLink)
	api.HandleFunc("GET", "/auth_redirect", h.authRedirect)

	api.HandleFunc("POST", "/login", h.login)
	api.HandleFunc("GET", "/auth_user", h.authUser)
	api.HandleFunc("GET", "/token", h.token)

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
	api.HandleFunc("GET", "/has_unread_notifications", h.hasUnreadNotifications)

	mime.AddExtensionType(".js", "application/javascript; charset=utf-8")
	fs := http.FileServer(&spaFileSystem{http.Dir("web/static")})
	if inLocalhost {
		fs = NoCache(fs)
	}

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", h.withAuth(api)))
	r.Handle("GET", "/...", fs)
	return r
}
