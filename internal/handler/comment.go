package handler

import (
	"encoding/json"
	"github.com/matryer/way"
	"mime"
	"net/http"
	"social-network/internal/service"
	"strconv"
)

type CreateCommentInput struct {
	Content string
}

func (h *handler) createComment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var in CreateCommentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	postID, _ := strconv.ParseInt(way.Param(ctx, "postID"), 10, 64)
	c, err := h.CreateComment(ctx, postID, in.Content)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err == service.ErrInvalidContnt {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err == service.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, c, http.StatusCreated)

}

func (h *handler) comments(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/stream" {
		h.subscribedToComments(w, r)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	postID, _ := strconv.ParseInt(way.Param(ctx, "postID"), 10, 64)
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)

	cc, err := h.Comments(ctx, postID, last, before)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, cc, http.StatusCreated)
}
func (h *handler) toggleCommentLike(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	commentID, _ := strconv.ParseInt(way.Param(ctx, "commentID"), 10, 64)
	out, err := h.ToggleCommentLike(ctx, commentID)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == service.ErrCommentNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, out, http.StatusOK)

}

func (h *handler) subscribedToComments(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondError(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	postID, _ := strconv.ParseInt(way.Param(ctx, "postID"), 10, 64)

	header := w.Header()
	header.Set("cache-control", "no-cache")
	header.Set("connection", "keep-alive")
	header.Set("content-type", "text/event-stream")

	for c := range h.SubscribedToComments(ctx, postID) {
		writeSSE(w, c)
		f.Flush()
	}
}
