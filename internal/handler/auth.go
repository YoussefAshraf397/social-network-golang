package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"social-network/internal/service"
	"strings"
)

type loginInput struct {
	Email string
}

type sendMagicLinkInput struct {
	Email       string
	RedirectURI string
}

func (h *handler) sendMagicLink(w http.ResponseWriter, r *http.Request) {

	var in sendMagicLinkInput
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.SendMagicLink(r.Context(), in.Email, in.RedirectURI)
	if err == service.ErrInvalidEmail || err == service.ErrInvalidRedirectURI {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) authRedirect(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	uri, err := h.AuthURI(r.Context(), q.Get("verification_code"), q.Get("redirect_uri"))
	if err == service.ErrInvalidVerificationCode || err == service.ErrInvalidRedirectURI {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == service.ErrVerificationCodeNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err == service.ErrVerificationCodeExpired {
		http.Error(w, err.Error(), http.StatusGone)
		return
	}
	if err != nil {
		respondError(w, err)
		return
	}

	http.Redirect(w, r, uri, http.StatusFound)
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.Login(r.Context(), in.Email)
	if err == service.ErrInvalidEmail {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, out, http.StatusOK)

}

func (h *handler) authUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.AuthUser(r.Context())
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, u, http.StatusOK)
}

func (h *handler) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := r.Header.Get("Authorization")
		if !strings.HasPrefix(a, "Bearer") {
			next.ServeHTTP(w, r)
			return
		}

		token := a[7:]
		uid, err := h.AuthUserId(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, service.KeyAuthUserId, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
