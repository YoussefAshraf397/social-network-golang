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
	api.HandleFunc("GET", "/users/:username", h.user)
	api.HandleFunc("GET", "/users", h.users)
	api.HandleFunc("POST", "/users/:username/toggle_follow", h.toggleFollow)
	api.HandleFunc("GET", "/users/:username/get_followers", h.followers)
	api.HandleFunc("GET", "/users/:username/get_followees", h.followees)

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", h.withAuth(api)))
	return r
}
