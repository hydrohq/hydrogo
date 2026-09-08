package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type httpUserRepo struct{ user *User }

func (r httpUserRepo) Create(context.Context, *User) error { return nil }
func (r httpUserRepo) GetByID(context.Context, string) (*User, error) {
	if r.user == nil {
		return nil, ErrUserNotFound
	}
	return r.user, nil
}
func (r httpUserRepo) GetByEmail(context.Context, string) (*User, error) {
	if r.user == nil {
		return nil, ErrUserNotFound
	}
	return r.user, nil
}
func (r httpUserRepo) Count(context.Context) (int64, error)          { return 1, nil }
func (r httpUserRepo) SetActive(context.Context, string, bool) error { return nil }

type httpSessionRepo struct{ session *Session }

func (r *httpSessionRepo) Create(_ context.Context, s *Session) error { r.session = s; return nil }
func (r *httpSessionRepo) GetByTokenHash(_ context.Context, tokenHash string) (*Session, error) {
	if r.session == nil || r.session.TokenHash != tokenHash {
		return nil, ErrSessionNotFound
	}
	return r.session, nil
}
func (r *httpSessionRepo) DeleteByTokenHash(context.Context, string) error {
	r.session = nil
	return nil
}
func (r *httpSessionRepo) DeleteByUserID(context.Context, string) error { r.session = nil; return nil }
func (r *httpSessionRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *httpSessionRepo) Touch(context.Context, string, time.Time) error { return nil }

type httpAudit struct{ actions []string }

func (a *httpAudit) Record(_ context.Context, action, _ string) error {
	a.actions = append(a.actions, action)
	return nil
}

func TestHTTPLoginCreatesSessionAndAudit(t *testing.T) {
	hash, err := HashPassword("secret", TestArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	sessions := &httpSessionRepo{}
	audit := &httpAudit{}
	service := NewAuthService(httpUserRepo{user: &User{ID: "u1", Email: "admin@example.com", PasswordHash: hash, Role: RoleAdmin, Active: true}}, sessions, TestArgon2Params)
	handler := HTTPHandler{Service: service, Audit: audit}.login

	form := url.Values{"email": {"admin@example.com"}, "password": {"secret"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if len(res.Result().Cookies()) != 1 || res.Result().Cookies()[0].Name != SessionCookieName {
		t.Fatalf("expected session cookie, got %v", res.Result().Cookies())
	}
	if len(audit.actions) != 1 || audit.actions[0] != "auth.login" {
		t.Fatalf("unexpected audit actions: %v", audit.actions)
	}
}
