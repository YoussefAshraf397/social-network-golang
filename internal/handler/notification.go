package handler

import (
	"github.com/matryer/way"
	"mime"
	"net/http"
	"social-network/internal/service"
	"strconv"
)

func (h *handler) notifications(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/stream" {
		h.subscribedToNotifications(w, r)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)

	nn, err := h.Notifications(ctx, last, before)

	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return

	}
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, nn, http.StatusOK)
}

func (h *handler) markNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	notificationID, _ := strconv.ParseInt(way.Param(ctx, "notificationID"), 10, 64)

	err := h.MarkNotificationAsRead(ctx, notificationID)

	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return

	}
	if err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) markAllNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := h.MarkAllNotificationAsRead(ctx)

	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return

	}
	if err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) subscribedToNotifications(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondError(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	nn, err := h.SubscribedToNotifications(ctx)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondError(w, err)
		return
	}

	header := w.Header()
	header.Set("cache-control", "no-cache")
	header.Set("connection", "keep-alive")
	header.Set("content-type", "text/event-stream")

	for n := range nn {
		writeSSE(w, n)
		f.Flush()
	}
}

func (h *handler) hasUnreadNotifications(w http.ResponseWriter, r *http.Request) {
	unread, err := h.HasUnreadNotifications(r.Context())
	if err != service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, unread, http.StatusOK)

}
