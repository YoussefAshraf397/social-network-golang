package handler

import (
	"net/http"
	"social-network/internal/service"
	"strconv"
)

func (h *handler) timeline(w http.ResponseWriter, r *http.Request) {
	//if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/stream" {
	//	h.subscribedToTimeline(w, r)
	//	return
	//}

	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	//log.Fatalln(before)
	tt, err := h.Timeline(ctx, last, before)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, tt, http.StatusOK)
}

func (h *handler) subscribedToTimeline(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondError(w, errStreamingUnsupported)
		return
	}

	tt, err := h.SubscribedToTimeline(r.Context())
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondError(w, err)
		return
	}
	header := w.Header()
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("Content-Type", "text/event-stream")

	for ti := range tt {
		writeSSE(w, ti)
		f.Flush()
	}

}
