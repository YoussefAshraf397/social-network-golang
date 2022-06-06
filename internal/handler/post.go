package handler

import (
	"encoding/json"
	"github.com/matryer/way"
	"net/http"
	"social-network/internal/service"
	"strconv"
)

type createPostInput struct {
	Content   string
	SpoilerOf *string
	NSFW      bool
}

func (h *handler) createpost(w http.ResponseWriter, r *http.Request) {
	var input createPostInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ti, err := h.CreatePost(r.Context(), input.Content, input.SpoilerOf, input.NSFW)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err == service.ErrInvalidContnt || err == service.ErrInvalidSpoiler {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, ti, http.StatusOK)
}

func (h *handler) togglePostLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID, _ := strconv.ParseInt(way.Param(ctx, "postID"), 10, 64)
	out, err := h.TogglePostLike(ctx, postID)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
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
	respond(w, out, http.StatusOK)
}

func (h *handler) posts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	pp, err := h.Posts(ctx, way.Param(ctx, "username"), last, before)
	if err == service.ErrInvalidUsername {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, pp, http.StatusOK)
}

func (h *handler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID, _ := strconv.ParseInt(way.Param(ctx, "postID"), 10, 64)

	p, err := h.Post(ctx, postID)
	if err == service.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, p, http.StatusOK)
}
