package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
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
func (r httpUserRepo) UpdateEmail(context.Context, database.Executor, string, string) error {
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

type loginSecurityTrackerStub struct {
	failures int
	previous time.Time
}

func (s *loginSecurityTrackerStub) RecordFailedLogin(context.Context, string) error {
	s.failures++
	return nil
}
func (s *loginSecurityTrackerStub) RecordSuccessfulLogin(context.Context, string, string, time.Time) (LoginSecurityNotice, error) {
	return LoginSecurityNotice{PreviousLoginAt: &s.previous, FailedAttempts: 2}, nil
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

type loginMFAStore struct{}

func (loginMFAStore) PutFactor(context.Context, database.Executor, string, []byte, bool) error {
	return nil
}
func (loginMFAStore) GetFactor(context.Context, database.Executor, string) ([]byte, bool, error) {
	return []byte("encrypted-factor"), true, nil
}
func (loginMFAStore) PutRecoveryCode(context.Context, database.Executor, string, string, string) error {
	return nil
}
func (loginMFAStore) ConsumeRecoveryCode(context.Context, database.Executor, string, string, time.Time) error {
	return nil
}

func TestHTTPLoginCreatesSessionAndAudit(t *testing.T) {
	hash, err := HashPassword("secret", TestArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	sessions := &httpSessionRepo{}
	audit := &httpAudit{}
	tracker := &loginSecurityTrackerStub{previous: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	service := NewAuthService(httpUserRepo{user: &User{ID: "u1", Email: "admin@example.com", PasswordHash: hash, Role: RoleAdmin, Active: true}}, sessions, nil, TestArgon2Params)
	handler := HTTPHandler{Service: service, Audit: audit, Secure: true, LoginSecurity: tracker}.login

	form := url.Values{"email": {"admin@example.com"}, "password": {"secret"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if len(res.Result().Cookies()) != 1 || res.Result().Cookies()[0].Name != SessionCookieName || !res.Result().Cookies()[0].Secure {
		t.Fatalf("expected session cookie, got %v", res.Result().Cookies())
	}
	if len(audit.actions) != 1 || audit.actions[0] != "auth.login" {
		t.Fatalf("unexpected audit actions: %v", audit.actions)
	}
	var response struct {
		SecurityNotice LoginSecurityNotice `json:"security_notice"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || response.SecurityNotice.FailedAttempts != 2 || response.SecurityNotice.PreviousLoginAt == nil {
		t.Fatalf("login security notice=%+v err=%v", response, err)
	}
	badForm := url.Values{"email": {"admin@example.com"}, "password": {"wrong"}}
	badRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(badForm.Encode()))
	badRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badResponse := httptest.NewRecorder()
	handler(badResponse, badRequest)
	if badResponse.Code != http.StatusUnauthorized || tracker.failures != 1 {
		t.Fatalf("failed login status=%d tracked=%d", badResponse.Code, tracker.failures)
	}
}

func TestHTTPLoginRequestsAuthenticatorCodeWithServerClock(t *testing.T) {
	hash, err := HashPassword("secret", TestArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	service := NewAuthService(httpUserRepo{user: &User{ID: "mfa-user", Email: "admin@example.com", PasswordHash: hash, Role: RoleAdmin, Active: true}}, &httpSessionRepo{}, nil, TestArgon2Params)
	handler := HTTPHandler{Service: service, MFA: &MFAService{Store: loginMFAStore{}}, MFADB: httpExecutor{}}.login
	form := url.Values{"email": {"admin@example.com"}, "password": {"secret"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("challenge status=%d body=%s", res.Code, res.Body.String())
	}
	var response struct {
		Error      string `json:"error"`
		Period     int64  `json:"totp_period_seconds"`
		ServerTime int64  `json:"server_time_unix"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "mfa_code_required" || response.Period != 30 || response.ServerTime == 0 {
		t.Fatalf("unexpected MFA challenge response: %+v", response)
	}
}

func TestSessionExtensionRotatesCookieWithoutPageReload(t *testing.T) {
	sessions := &httpSessionRepo{}
	service := NewAuthService(httpUserRepo{}, sessions, nil, TestArgon2Params)
	handler := HTTPHandler{Service: service, Secure: true}
	user := &User{ID: "user-extend", Email: "editor@example.com", Role: RoleEditor, Active: true}
	current := &Session{ID: "old-session", UserID: user.ID, ExpiresAt: time.Now().Add(time.Minute)}
	req := httptest.NewRequest(http.MethodPost, "/auth/me/session/extend", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "old-token"})
	req = req.WithContext(WithUser(WithSession(req.Context(), current), user))
	recorder := httptest.NewRecorder()
	handler.extendSession(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var cookie *http.Cookie
	for _, candidate := range recorder.Result().Cookies() {
		if candidate.Name == SessionCookieName {
			cookie = candidate
		}
	}
	if cookie == nil || cookie.Value == "old-token" || !cookie.Secure {
		t.Fatalf("session cookie was not safely rotated: cookie=%+v", cookie)
	}
}

func TestHTTPHandlerRequiresMFAForConfiguredRole(t *testing.T) {
	handler := HTTPHandler{MFARequiredRoles: []Role{RoleAdmin}}
	if !handler.requiresMFA(RoleAdmin) || handler.requiresMFA(RoleEditor) {
		t.Fatal("MFA role policy did not match configured role")
	}
}

func TestHTTPLoginRejectsOversizedForm(t *testing.T) {
	hash, err := HashPassword("secret", TestArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	service := NewAuthService(httpUserRepo{user: &User{ID: "u1", Email: "admin@example.com", PasswordHash: hash, Active: true}}, &httpSessionRepo{}, nil, TestArgon2Params)
	handler := HTTPHandler{Service: service}.login

	form := url.Values{"email": {"admin@example.com"}, "password": {strings.Repeat("x", maxAuthFormBytes)}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("oversized login status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestHTTPLoginMapsMissingUserRepositoryToUnavailable(t *testing.T) {
	service := NewAuthService(nil, &httpSessionRepo{}, nil, TestArgon2Params)
	handler := HTTPHandler{Service: service}.login
	form := url.Values{"email": {"user@example.com"}, "password": {"secret"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing user repository status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestHTTPLoginRejectsTypedNilMFAExecutorAsUnavailable(t *testing.T) {
	hash, err := HashPassword("secret", TestArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	var db *httpExecutor
	service := NewAuthService(httpUserRepo{user: &User{
		ID: "u1", Email: "user@example.com", PasswordHash: hash, Active: true,
	}}, &httpSessionRepo{}, nil, TestArgon2Params)
	handler := HTTPHandler{
		Service: service,
		MFA:     &MFAService{},
		MFADB:   db,
	}.login
	form := url.Values{"email": {"user@example.com"}, "password": {"secret"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	handler(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("typed nil MFA executor status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestValidateSessionRejectsAbsoluteExpiryBeforeAdapter(t *testing.T) {
	sessions := &httpSessionRepo{session: &Session{
		ID:         "s1",
		UserID:     "u1",
		TokenHash:  HashSessionToken("expired-token"),
		ExpiresAt:  time.Now().UTC().Add(-time.Minute),
		LastSeenAt: time.Now().UTC(),
	}}
	service := NewAuthService(httpUserRepo{user: &User{ID: "u1", Active: true}}, sessions, nil, TestArgon2Params)

	if _, _, err := service.ValidateSession(context.Background(), "expired-token"); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("ValidateSession error = %v, want %v", err, ErrSessionExpired)
	}
	if sessions.session != nil {
		t.Fatal("expired session was not revoked")
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

func TestPasswordResetRequestIsRateLimited(t *testing.T) {
	user := &User{ID: "u1", Email: "admin@example.com", Active: true}
	service := &PasswordRecovery{Users: httpUserRepo{user: user}, Tokens: httpRecoveryStore{}, Sessions: &httpSessionRepo{}, ArgonParams: TestArgon2Params}
	handler := HTTPHandler{
		Recovery:        service,
		RecoveryDB:      httpExecutor{},
		RecoveryLimiter: security.NewAttemptLimiter(1, 1, time.Hour),
		Notifier: RecoveryNotifierFunc(func(_ context.Context, _, _ string) error {
			return nil
		}),
	}.requestPasswordReset
	form := url.Values{"email": {user.Email}}
	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/auth/password-reset/request", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}
	first := httptest.NewRecorder()
	handler(first, newRequest())
	if first.Code != http.StatusAccepted {
		t.Fatalf("first reset status = %d", first.Code)
	}
	second := httptest.NewRecorder()
	handler(second, newRequest())
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second reset status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
}

func TestPasswordResetConfirmIsRateLimitedByClient(t *testing.T) {
	service := &PasswordRecovery{Users: httpUserRepo{user: &User{ID: "u1", Active: true}}, Tokens: httpRecoveryStore{}, Sessions: &httpSessionRepo{}, ArgonParams: TestArgon2Params}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := HTTPHandler{
		Recovery:               service,
		Transactions:           database.NewTransactor(db),
		RecoveryConfirmLimiter: security.NewAttemptLimiter(1, 1, time.Hour),
	}.confirmPasswordReset
	form := url.Values{"token": {"invalid-token"}, "password": {"Amber-River7!Cedar"}}
	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/auth/password-reset/confirm", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}
	first := httptest.NewRecorder()
	handler(first, newRequest())
	if first.Code != http.StatusNoContent {
		t.Fatalf("first reset confirm status = %d", first.Code)
	}
	second := httptest.NewRecorder()
	handler(second, newRequest())
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second reset confirm status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
}

func TestAdminInviteRequestRequiresAdminAndNotifiesToken(t *testing.T) {
	service := &InvitationService{Users: httpUserRepo{}, Invites: httpRecoveryStore{}, ArgonParams: TestArgon2Params}
	var notified string
	handler := HTTPHandler{
		Invitation:   service,
		InviteDB:     httpExecutor{},
		PublicOrigin: "https://hydro.example.test",
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
	var response map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["email"] != "new@example.com" || !strings.Contains(response["url"], "https://hydro.example.test/auth/invites/accept?token=") || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("share response = %#v, headers=%v", response, res.Header())
	}
}

func TestInviteAcceptanceIsRateLimitedByClient(t *testing.T) {
	service := &InvitationService{Users: httpUserRepo{}, Invites: httpRecoveryStore{}, ArgonParams: TestArgon2Params}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := HTTPHandler{
		Invitation:          service,
		Transactions:        database.NewTransactor(db),
		InviteAcceptLimiter: security.NewAttemptLimiter(1, 1, time.Hour),
	}.acceptInvite
	form := url.Values{"token": {"invalid-token"}, "password": {"Amber-River7!Cedar"}}
	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/auth/invites/accept", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}
	first := httptest.NewRecorder()
	handler(first, newRequest())
	if first.Code != http.StatusUnauthorized {
		t.Fatalf("first invite acceptance status = %d", first.Code)
	}
	second := httptest.NewRecorder()
	handler(second, newRequest())
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second invite acceptance status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
}

func TestAdminPermissionEndpointAcceptsOnlyEditionCatalogNames(t *testing.T) {
	catalog, err := NewPermissionCatalog("article.read")
	if err != nil {
		t.Fatal(err)
	}
	store := &permissionTestStore{}
	service := NewAuthService(httpUserRepo{}, &httpSessionRepo{}, nil, TestArgon2Params)
	service.SetPermissionStore(store)
	if err := service.SetPermissionCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	values := url.Values{"perm_article.read": {"allow"}, "perm_unregistered.delete": {"allow"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/users/user-2/permissions", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "user-2")
	req = req.WithContext(WithUser(req.Context(), &User{ID: "admin", Role: RoleAdmin, Active: true}))
	rec := httptest.NewRecorder()
	(HTTPHandler{Service: service}).updateUserPermissions(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("permission update status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(store.saved) != 1 || !store.saved["article.read"] {
		t.Fatalf("saved permissions included undeclared names: %#v", store.saved)
	}
}

func TestAdminPermissionEndpointBoundsFormBody(t *testing.T) {
	service := NewAuthService(httpUserRepo{}, &httpSessionRepo{}, nil, TestArgon2Params)
	service.SetPermissionStore(&permissionTestStore{})
	req := httptest.NewRequest(http.MethodPost, "/auth/users/user-2/permissions", strings.NewReader(strings.Repeat("x", maxPermissionFormBytes+1)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "user-2")
	req = req.WithContext(WithUser(req.Context(), &User{ID: "admin", Role: RoleAdmin, Active: true}))
	rec := httptest.NewRecorder()
	(HTTPHandler{Service: service}).updateUserPermissions(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized permission form status=%d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLoginRateLimitBypassByChangingEmail(t *testing.T) {
	service := &AuthService{userRepo: httpUserRepo{}}
	handler := HTTPHandler{
		Service:      service,
		LoginLimiter: security.NewAttemptLimiter(1, 1, time.Minute),
	}.login

	makeRequest := func(email string) *httptest.ResponseRecorder {
		form := url.Values{"email": {email}, "password": {"wrong"}}
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}

	first := makeRequest("user1@example.com")
	if first.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", first.Code)
	}

	second := makeRequest("user1@example.com")
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected 429, got %d", second.Code)
	}

	third := makeRequest("user2@example.com")
	if third.Code == http.StatusUnauthorized {
		t.Fatalf("VULNERABILITY: Rate limit bypassed by changing email! Expected 429, got 401")
	} else if third.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected 429, got %d", third.Code)
	}
}
