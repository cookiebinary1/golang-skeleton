// Package handler exposes the demo REST API of the api service.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/cookiebinary1/golang-skeleton/services/api/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// UserStore is the persistence contract used by HTTP handlers.
type UserStore interface {
	Create(context.Context, store.User) (store.User, error)
	List(context.Context) ([]store.User, error)
	GetByEmail(context.Context, string) (store.User, error)
}

type Handler struct {
	log  *slog.Logger
	repo UserStore
}

func New(log *slog.Logger, repo UserStore) *Handler {
	return &Handler{log: log, repo: repo}
}

// Routes registers api endpoints on the shared router.
func (h *Handler) Routes(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/users", h.createUser)
		r.Get("/users", h.listUsers)
		r.Get("/users/{email}", h.getUser)
	})
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		httpError(w, http.StatusBadRequest, "expected a single JSON object")
		return
	}
	req.Name, req.Email = strings.TrimSpace(req.Name), strings.TrimSpace(req.Email)
	address, parseErr := mail.ParseAddress(req.Email)
	if req.Name == "" || parseErr != nil || address.Address != req.Email {
		httpError(w, http.StatusBadRequest, "name and email are required")
		return
	}

	u, err := h.repo.Create(r.Context(), store.User{Name: req.Name, Email: req.Email})
	switch {
	case errors.Is(err, store.ErrDuplicateEmail):
		httpError(w, http.StatusConflict, "email already exists")
		return
	case err != nil:
		h.log.Error("create user failed", "err", err)
		httpError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, u)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.List(r.Context())
	if err != nil {
		h.log.Error("list users failed", "err", err)
		httpError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")
	// Chi matches RawPath when present; otherwise URL.Path is already decoded.
	// Unescaping only the raw form avoids decoding literal percent sequences twice.
	if r.URL.RawPath != "" {
		decoded, err := url.PathUnescape(email)
		if err != nil {
			httpError(w, http.StatusBadRequest, "invalid email path")
			return
		}
		email = decoded
	}
	u, err := h.repo.GetByEmail(r.Context(), email)
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpError(w, http.StatusNotFound, "user not found")
		return
	case err != nil:
		h.log.Error("get user failed", "err", err)
		httpError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
