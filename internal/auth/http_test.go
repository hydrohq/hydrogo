package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	_ "modernc.org/sqlite"
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
func (r httpUserRepo) UpdatePassword(context.Context, database.Executor, string, string) error {
	return nil
}

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

type httpRecoveryStore struct{}

func (httpRecoveryStore) CreateRecoveryToken(context.Context, database.Executor, string, string, string, string, time.Time, time.Time) error {
	return nil
}
func (httpRecoveryStore) ConsumeRecoveryToken(context.Context, database.Executor, string, string, time.Time) (string, error) {
	return "u1", nil
}

func (httpRecoveryStore) CreateInvite(context.Context, database.Executor, string, string, Role, string, time.Time, time.Time) error {
	return nil
}
func (httpRecoveryStore) ConsumeInvite(context.Context, database.Executor, string, time.Time) (string, Role, error) {
	return "", RoleEditor, ErrInviteInvalid
}

type httpExecutor struct{}

func (httpExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) { return nil, nil }
func (httpExecutor) QueryRowContext(context.Context, string, ...any) *sql.Row        { return nil }

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

func TestPasswordResetRequestDoesNotExposeAccountState(t *testing.T) {
	user := &User{ID: "u1", Email: "admin@example.com", Active: true}
	service := &PasswordRecovery{Users: httpUserRepo{user: user}, Tokens: httpRecoveryStore{}, Sessions: &httpSessionRepo{}, ArgonParams: TestArgon2Params}
	var notifiedEmail, notifiedToken string
	handler := HTTPHandler{
		Recovery:   service,
		RecoveryDB: httpExecutor{},
		Notifier: RecoveryNotifierFunc(func(_ context.Context, email, token string) error {
			notifiedEmail, notifiedToken = email, token
			return nil
		}),
	}.requestPasswordReset
	form := url.Values{"email": {user.Email}}
	req := httptest.NewRequest(http.MethodPost, "/auth/password-reset/request", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler(res, req)
	if res.Code != http.StatusAccepted || notifiedEmail != user.Email || notifiedToken == "" {
		t.Fatalf("reset request = %d, email=%q token=%q", res.Code, notifiedEmail, notifiedToken)
	}
	unknown := httptest.NewRequest(http.MethodPost, "/auth/password-reset/request", strings.NewReader(url.Values{"email": {"missing@example.com"}}.Encode()))
	unknown.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	unknownRes := httptest.NewRecorder()
	handler(unknownRes, unknown)
	if unknownRes.Code != http.StatusAccepted {
		t.Fatalf("unknown reset status = %d", unknownRes.Code)
	}
}

func TestAdminInviteRequestRequiresAdminAndNotifiesToken(t *testing.T) {
	service := &InvitationService{Users: httpUserRepo{}, Invites: httpRecoveryStore{}, ArgonParams: TestArgon2Params}
	var notified string
	handler := HTTPHandler{
		Invitation: service,
		InviteDB:   httpExecutor{},
		InviteNotice: InviteNotifierFunc(func(_ context.Context, email, token string) error {
			notified = email + ":" + token
			return nil
		}),
	}.issueInvite
	form := url.Values{"email": {"new@example.com"}, "role": {string(RoleEditor)}}
	req := httptest.NewRequest(http.MethodPost, "/admin/auth/invites", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(WithUser(req.Context(), &User{ID: "admin", Role: RoleAdmin, Active: true}))
	res := httptest.NewRecorder()
	handler(res, req)
	if res.Code != http.StatusAccepted || !strings.HasPrefix(notified, "new@example.com:") {
		t.Fatalf("invite request = %d, notification=%q", res.Code, notified)
	}
}
