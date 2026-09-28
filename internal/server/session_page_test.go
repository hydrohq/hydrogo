package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type sessionPageRepo struct {
	items     []auth.Session
	revoked   bool
	revokedID string
}

func (*sessionPageRepo) Create(context.Context, *auth.Session) error { return nil }
func (*sessionPageRepo) GetByTokenHash(context.Context, string) (*auth.Session, error) {
	return nil, auth.ErrSessionNotFound
}
func (*sessionPageRepo) DeleteByTokenHash(context.Context, string) error         { return nil }
func (*sessionPageRepo) DeleteByUserID(context.Context, string) error            { return nil }
func (*sessionPageRepo) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }
func (*sessionPageRepo) Touch(context.Context, string, time.Time) error          { return nil }
func (r *sessionPageRepo) ListByUserID(context.Context, string, time.Time) ([]auth.Session, error) {
	return r.items, nil
}
func (r *sessionPageRepo) DeleteOtherByUserID(context.Context, string, string) error {
	r.revoked = true
	return nil
}
func (r *sessionPageRepo) DeleteByIDAndUserID(_ context.Context, _, id string) error {
	r.revokedID = id
	return nil
}

func TestSessionPageListsCurrentAndRevokesOtherSessions(t *testing.T) {
	repository := &sessionPageRepo{items: []auth.Session{{ID: "current", UserAgent: "Browser", IPAddress: "192.0.2.1", LastSeenAt: time.Now()}, {ID: "other", UserAgent: "Phone", IPAddress: "192.0.2.2"}}}
	handler := SessionPageHandler{Service: auth.NewAuthService(nil, repository, nil, auth.TestArgon2Params)}
	ctx := auth.WithSession(auth.WithUser(context.Background(), &auth.User{ID: "user", Active: true}), &auth.Session{ID: "current", UserID: "user", TokenHash: "hash"})
	get := httptest.NewRequest(http.MethodGet, "/auth/me/sessions/manage", nil).WithContext(ctx)
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, get)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Current device") || !strings.Contains(page.Body.String(), "Phone") {
		t.Fatalf("GET status=%d body=%s", page.Code, page.Body.String())
	}
	post := httptest.NewRequest(http.MethodPost, "/auth/me/sessions/manage/revoke-others", strings.NewReader("_csrf=token"))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post = post.WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, post)
	if response.Code != http.StatusSeeOther || !repository.revoked {
		t.Fatalf("POST status=%d revoked=%v", response.Code, repository.revoked)
	}
	revoke := httptest.NewRequest(http.MethodPost, "/auth/me/sessions/manage/other/revoke", nil).WithContext(ctx)
	revoke.SetPathValue("id", "other")
	revokeResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeResponse, revoke)
	if revokeResponse.Code != http.StatusSeeOther || repository.revokedID != "other" {
		t.Fatalf("individual revoke status=%d session=%q", revokeResponse.Code, repository.revokedID)
	}
}
