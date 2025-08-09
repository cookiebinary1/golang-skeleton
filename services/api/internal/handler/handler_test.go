package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cookiebinary1/golang-skeleton/services/api/internal/store"
	"github.com/go-chi/chi/v5"
)

func TestRejectInvalidCreate(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := chi.NewRouter()
	New(log, nil).Routes(router)
	for _, body := range []string{
		`{`, `{}`, `{"name":" ","email":"a@example.com"}`,
		`{"name":"Alice","email":"invalid"}`,
		`{"name":"Alice","email":"Alice <a@example.com>"}`,
		`{"name":"Alice","email":"a@example.com","unknown":true}`,
		`{"name":"Alice","email":"a@example.com"} {}`,
		`{"name":"` + strings.Repeat("a", 1<<20) + `","email":"a@example.com"}`,
	} {
		req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != 400 {
			t.Errorf("invalid request status = %d, want 400", resp.Code)
		}
	}
}

type emailStore struct{ got string }

func (s *emailStore) Create(context.Context, store.User) (store.User, error) {
	panic("unexpected Create")
}
func (s *emailStore) List(context.Context) ([]store.User, error) { panic("unexpected List") }
func (s *emailStore) GetByEmail(_ context.Context, email string) (store.User, error) {
	s.got = email
	return store.User{ID: 1, Name: "Test", Email: email, CreatedAt: time.Now()}, nil
}

func TestGetUserDecodesEmailExactlyOnce(t *testing.T) {
	cases := []struct{ path, email string }{
		{"alex@example.com", "alex@example.com"},
		{url.PathEscape("alex@example.com"), "alex@example.com"},
		{url.PathEscape("alex+team@example.com"), "alex+team@example.com"},
		{url.PathEscape("alex%40team@example.com"), "alex%40team@example.com"},
		{url.PathEscape("alex/team@example.com"), "alex/team@example.com"},
	}
	for _, tt := range cases {
		t.Run(tt.path, func(t *testing.T) {
			repo := &emailStore{}
			router := chi.NewRouter()
			New(slog.New(slog.NewTextHandler(io.Discard, nil)), repo).Routes(router)
			req := httptest.NewRequest("GET", "/api/v1/users/"+tt.path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != 200 {
				t.Fatalf("status %d, want 200", response.Code)
			}
			if repo.got != tt.email {
				t.Errorf("got %q, want %q", repo.got, tt.email)
			}
		})
	}
}
