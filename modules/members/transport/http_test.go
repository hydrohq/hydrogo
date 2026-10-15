package transport_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	memberstore "github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/members/transport"
	_ "modernc.org/sqlite"
)

type memoryMemberStore struct {
	members map[string]*members.Member
	byEmail map[string]*members.Member
}

type trackedMemberStore struct {
	*memoryMemberStore
	activeWrites   int
	verifiedWrites int
}

type fixedMemberStore struct {
	*memoryMemberStore
	rows []members.Member
}

type verificationNotifier struct{ email, link string }

func (n *verificationNotifier) SendEmailVerification(_ context.Context, email, link string) error {
	n.email, n.link = email, link
	return nil
}

func (s fixedMemberStore) Search(_ context.Context, filter members.MemberFilter) (members.MemberListResult, error) {
	start := filter.Offset
	if start > len(s.rows) {
		start = len(s.rows)
	}
	end := start + filter.Limit
	if end > len(s.rows) {
		end = len(s.rows)
	}
	return members.MemberListResult{Members: s.rows[start:end], TotalCount: len(s.rows)}, nil
}

func newMemoryMemberStore() *memoryMemberStore {
	return &memoryMemberStore{
		members: make(map[string]*members.Member),
		byEmail: make(map[string]*members.Member),
	}
}

func (m *memoryMemberStore) Create(_ context.Context, member *members.Member) error {
	m.members[member.ID] = member
	m.byEmail[member.Email] = member
	return nil
}

func (m *memoryMemberStore) GetByID(_ context.Context, id string) (*members.Member, error) {
	if mem, ok := m.members[id]; ok {
		return mem, nil
	}
	return nil, members.ErrMemberNotFound
}

func (m *memoryMemberStore) GetByEmail(_ context.Context, email string) (*members.Member, error) {
	if mem, ok := m.byEmail[email]; ok {
		return mem, nil
	}
	return nil, members.ErrMemberNotFound
}

func (m *memoryMemberStore) List(_ context.Context, _ int) ([]members.Member, error) {
	return nil, nil
}

func (m *memoryMemberStore) Search(_ context.Context, _ members.MemberFilter) (members.MemberListResult, error) {
	return members.MemberListResult{}, nil
}

func (m *memoryMemberStore) SetActive(_ context.Context, id string, active bool) error {
	if member, ok := m.members[id]; ok {
		member.Active = active
	}
	return nil
}

func (m *memoryMemberStore) SetEmailVerified(_ context.Context, _ string, _ bool) error {
	return nil
}

func (m *trackedMemberStore) SetActive(ctx context.Context, id string, active bool) error {
	m.activeWrites++
	return m.memoryMemberStore.SetActive(ctx, id, active)
}

func (m *trackedMemberStore) SetEmailVerified(_ context.Context, _ string, verified bool) error {
	m.verifiedWrites++
	if member, ok := m.members["member-1"]; ok {
		if verified {
			now := time.Now()
			member.EmailVerifiedAt = &now
		} else {
			member.EmailVerifiedAt = nil
		}
	}
	return nil
}

func (m *memoryMemberStore) UpdateLastLogin(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (m *memoryMemberStore) Stats(_ context.Context) (members.MemberStats, error) {
	return members.MemberStats{}, nil
}

type memorySessionStore struct {
	sessions map[string]*members.Session
}

func newMemorySessionStore() *memorySessionStore {
	return &memorySessionStore{sessions: make(map[string]*members.Session)}
}

func (s *memorySessionStore) Create(_ context.Context, session *members.Session) error {
	s.sessions[session.TokenHash] = session
	return nil
}

func (s *memorySessionStore) GetByTokenHash(_ context.Context, hash string) (*members.Session, error) {
	if sess, ok := s.sessions[hash]; ok {
		return sess, nil
	}
	return nil, members.ErrMemberNotFound
}

func (s *memorySessionStore) DeleteByTokenHash(_ context.Context, hash string) error {
	delete(s.sessions, hash)
	return nil
}

func (s *memorySessionStore) DeleteByID(_ context.Context, sessionID string) error {
	for k, v := range s.sessions {
		if v.ID == sessionID {
			delete(s.sessions, k)
			return nil
		}
	}
	return nil
}

func (s *memorySessionStore) DeleteByMemberID(_ context.Context, memberID string) error {
	for k, v := range s.sessions {
		if v.MemberID == memberID {
			delete(s.sessions, k)
		}
	}
	return nil
}

func (s *memorySessionStore) GetByID(_ context.Context, memberID, sessionID string) (*members.Session, error) {
	for _, session := range s.sessions {
		if session.MemberID == memberID && session.ID == sessionID {
			return session, nil
		}
	}
	return nil, members.ErrMemberNotFound
}

// ListByMemberID mirrors sqlstore and returns at most the 100 most recent sessions.
func (s *memorySessionStore) ListByMemberID(_ context.Context, memberID string) ([]members.Session, error) {
	var result []members.Session
	for _, session := range s.sessions {
		if session.MemberID == memberID {
			result = append(result, *session)
		}
	}
	if len(result) > 100 {
		result = result[:100]
	}
	return result, nil
}

func (s *memorySessionStore) Touch(_ context.Context, hash string, t time.Time) error {
	if sess, ok := s.sessions[hash]; ok {
		sess.LastSeenAt = t
	}
	return nil
}

func TestMemberTransportRegistrationAndLogin(t *testing.T) {
	svc := members.Service{
		Members:     newMemoryMemberStore(),
		Sessions:    newMemorySessionStore(),
		ArgonParams: auth.DefaultArgon2Params,
	}
	handler := transport.Handler{Service: svc}

	mux := http.NewServeMux()
	for _, r := range handler.Routes() {
		mux.Handle(r.Pattern, r.Handler)
	}

	// 1. POST /member/register creates account and sets cookie
	form := url.Values{"email": {"newuser@example.com"}, "password": {"Amber-River7!Cedar"}}
	postReq := httptest.NewRequest("POST", "/member/register", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRR := httptest.NewRecorder()
	mux.ServeHTTP(postRR, postReq)
	if postRR.Code != http.StatusOK {
		t.Fatalf("POST /member/register status = %d, body = %s", postRR.Code, postRR.Body.String())
	}
	cookies := postRR.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == members.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("session cookie not set on registration")
	}

	// 2. GET /member/me with session cookie returns 200 OK and email in JSON
	meReq := httptest.NewRequest("GET", "/member/me", nil)
	meReq.AddCookie(sessionCookie)
	meRR := httptest.NewRecorder()
	mux.ServeHTTP(meRR, meReq)
	if meRR.Code != http.StatusOK {
		t.Fatalf("GET /member/me status = %d", meRR.Code)
	}
	if !strings.Contains(meRR.Body.String(), `"email":"newuser@example.com"`) {
		t.Fatalf("GET /member/me JSON missing email: %s", meRR.Body.String())
	}

	// 3. GET /member/me without session cookie returns 401 Unauthorized
	unauthReq := httptest.NewRequest("GET", "/member/me", nil)
	unauthRR := httptest.NewRecorder()
	mux.ServeHTTP(unauthRR, unauthReq)
	if unauthRR.Code != http.StatusUnauthorized {
		t.Fatalf("GET /member/me unauthenticated status = %d, expected 401", unauthRR.Code)
	}
}

func TestAdminDeactivateRevokesBrowserSessions(t *testing.T) {
	memberStore := newMemoryMemberStore()
	memberStore.members["member-1"] = &members.Member{ID: "member-1", Email: "member@example.com", Active: true}
	sessions := newMemorySessionStore()
	svc := members.Service{Members: memberStore, Sessions: sessions}
	token, session, err := svc.CreateSession(context.Background(), "member-1", "test", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handler := transport.Handler{Service: svc}
	request := httptest.NewRequest(http.MethodPatch, transport.AdminBasePath+"/member-1", strings.NewReader(`{"active":false}`))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"active":false`) {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if _, _, err := svc.ValidateSession(context.Background(), token); !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("deactivated member session still valid: session=%v err=%v", session.ID, err)
	}
}

func TestAdminListDetailAndStatsAnswerEnvelopes(t *testing.T) {
	now := time.Now().UTC()
	rows := []members.Member{{ID: "m-1", Email: "a@example.test", Active: true, CreatedAt: now}, {ID: "m-2", Email: "b@example.test", Active: false, CreatedAt: now}, {ID: "m-3", Email: "c@example.test", Active: true, CreatedAt: now}}
	store := fixedMemberStore{memoryMemberStore: newMemoryMemberStore(), rows: rows}
	for i := range rows {
		store.members[rows[i].ID] = &rows[i]
	}
	svc := members.Service{Members: store, Sessions: newMemorySessionStore()}
	mux := http.NewServeMux()
	for _, route := range (transport.Handler{Service: svc}).Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	get := func(path string, user *auth.User) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	list := get(transport.AdminBasePath+"?limit=2&offset=0", admin)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"total":3`) || !strings.Contains(list.Body.String(), `"has_more":true`) || strings.Contains(list.Body.String(), "c@example.test") {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	if bad := get(transport.AdminBasePath+"?status=banned", admin); bad.Code != http.StatusBadRequest || !strings.Contains(bad.Body.String(), `"status":"invalid"`) {
		t.Fatalf("invalid status filter: %d %s", bad.Code, bad.Body.String())
	}
	detail := get(transport.AdminBasePath+"/m-2", admin)
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"member":{"id":"m-2"`) || !strings.Contains(detail.Body.String(), `"sessions":[]`) {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	if missing := get(transport.AdminBasePath+"/nope", admin); missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"code":"not_found"`) {
		t.Fatalf("missing member: %d %s", missing.Code, missing.Body.String())
	}
	if stats := get(transport.AdminBasePath+"/stats", admin); stats.Code != http.StatusOK || !strings.Contains(stats.Body.String(), `"total_members"`) {
		t.Fatalf("stats: %d %s", stats.Code, stats.Body.String())
	}
	export := get(transport.AdminBasePath+"/export?format=csv", admin)
	if export.Code != http.StatusOK || !strings.HasPrefix(export.Header().Get("Content-Type"), "text/csv") || !strings.Contains(export.Body.String(), "a@example.test,true") {
		t.Fatalf("export: %d %q %s", export.Code, export.Header().Get("Content-Type"), export.Body.String())
	}
	if viewer := get(transport.AdminBasePath, &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}); viewer.Code != http.StatusForbidden {
		t.Fatalf("viewer list status=%d", viewer.Code)
	}
}

func TestAdminRevokeSessionBindsSessionToMember(t *testing.T) {
	memberStore := newMemoryMemberStore()
	memberStore.members["member-a"] = &members.Member{ID: "member-a", Email: "a@example.com", Active: true}
	memberStore.members["member-b"] = &members.Member{ID: "member-b", Email: "b@example.com", Active: true}
	sessions := newMemorySessionStore()
	svc := members.Service{Members: memberStore, Sessions: sessions}
	tokenA, sessionA, err := svc.CreateSession(context.Background(), "member-a", "test", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tokenB, sessionB, err := svc.CreateSession(context.Background(), "member-b", "test", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handler := transport.Handler{Service: svc}
	var revoke http.Handler
	for _, route := range handler.Routes() {
		if route.Pattern == "POST "+transport.AdminBasePath+"/{id}/sessions/{session_id}/revoke" {
			revoke = route.Handler
		}
	}
	request := httptest.NewRequest(http.MethodPost, transport.AdminBasePath+"/member-a/sessions/"+sessionB.ID+"/revoke", nil)
	request.SetPathValue("id", "member-a")
	request.SetPathValue("session_id", sessionB.ID)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	revoke.ServeHTTP(response, request)
	if _, _, err := svc.ValidateSession(context.Background(), tokenB); err != nil {
		t.Fatalf("member B session was revoked: %v", err)
	}
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-member revoke status = %d, want %d", response.Code, http.StatusNotFound)
	}
	request = httptest.NewRequest(http.MethodPost, transport.AdminBasePath+"/member-a/sessions/"+sessionA.ID+"/revoke", nil)
	request.SetPathValue("id", "member-a")
	request.SetPathValue("session_id", sessionA.ID)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response = httptest.NewRecorder()
	revoke.ServeHTTP(response, request)
	if _, _, err := svc.ValidateSession(context.Background(), tokenA); !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("member A session survived own revoke: %v", err)
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("own revoke status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestAdminResendsAndConsumesOneUseMemberVerificationLink(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE members (id TEXT PRIMARY KEY,email TEXT NOT NULL UNIQUE,password_hash TEXT NOT NULL,active INTEGER NOT NULL,email_verified_at TEXT,last_login_at TEXT,created_at TEXT NOT NULL,updated_at TEXT NOT NULL);
CREATE TABLE member_email_verification_tokens (id TEXT PRIMARY KEY,member_id TEXT NOT NULL,token_hash TEXT NOT NULL UNIQUE,expires_at TEXT NOT NULL,consumed_at TEXT,created_at TEXT NOT NULL,tenant_id TEXT NOT NULL DEFAULT '');`)
	if err != nil {
		t.Fatal(err)
	}
	store := memberstore.Store{DB: db, Dialect: database.SQLite}
	now := time.Now().UTC()
	if err := store.Create(context.Background(), &members.Member{ID: "member-1", Email: "member@example.test", PasswordHash: "hash", Active: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	notifier := &verificationNotifier{}
	service := members.EmailVerificationService{Tokens: store}
	handler := transport.Handler{
		Service: members.Service{Members: store}, DB: db,
		Verification: service, VerificationTx: database.NewTenantTransactor(db, database.SQLite, false),
		VerificationNotifier: notifier, PublicOrigin: "https://hydro.example.test", VerificationEnabled: true,
	}
	var resend http.Handler
	var verify http.Handler
	for _, route := range handler.Routes() {
		if route.Pattern == "POST "+transport.AdminBasePath+"/{id}/resend-verification" {
			resend = route.Handler
		}
		if route.Pattern == "GET /member/email/verify" {
			verify = route.Handler
		}
	}
	request := httptest.NewRequest(http.MethodPost, transport.AdminBasePath+"/member-1/resend-verification", nil)
	request.SetPathValue("id", "member-1")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	resend.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || notifier.email != "member@example.test" || !strings.HasPrefix(notifier.link, "https://hydro.example.test/member/email/verify?token=") {
		t.Fatalf("resend status=%d notification=%+v body=%s", response.Code, notifier, response.Body.String())
	}
	parsed, err := url.Parse(notifier.link)
	if err != nil {
		t.Fatal(err)
	}
	verifyRequest := httptest.NewRequest(http.MethodGet, "/member/email/verify?token="+url.QueryEscape(parsed.Query().Get("token")), nil)
	verifyResponse := httptest.NewRecorder()
	verify.ServeHTTP(verifyResponse, verifyRequest)
	if verifyResponse.Code != http.StatusSeeOther || verifyResponse.Header().Get("Location") != "/member/login?email_verified=1" {
		t.Fatalf("verify status=%d location=%q", verifyResponse.Code, verifyResponse.Header().Get("Location"))
	}
	member, err := store.GetByID(context.Background(), "member-1")
	if err != nil || member.EmailVerifiedAt == nil {
		t.Fatalf("member verification state=%+v err=%v", member, err)
	}
	replay := httptest.NewRecorder()
	verify.ServeHTTP(replay, verifyRequest)
	if replay.Code != http.StatusBadRequest {
		t.Fatalf("verification replay status=%d, want 400", replay.Code)
	}
}

func TestMemberRegisterRejectsOversizedForm(t *testing.T) {
	svc := members.Service{
		Members:     newMemoryMemberStore(),
		Sessions:    newMemorySessionStore(),
		ArgonParams: auth.DefaultArgon2Params,
	}
	handler := transport.Handler{Service: svc}

	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}

	padding := strings.Repeat("a", 256<<10)
	form := url.Values{
		"email":    {"oversize@example.com"},
		"password": {"Amber-River7!Cedar"},
		"padding":  {padding},
	}
	req := httptest.NewRequest(http.MethodPost, "/member/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(form.Encode()))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("POST /member/register accepted oversized form (%d padding bytes)", len(padding))
	}
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("POST /member/register status=%d, want 413", rr.Code)
	}
}

func TestMemberLoginRejectsOversizedForm(t *testing.T) {
	svc := members.Service{
		Members:     newMemoryMemberStore(),
		Sessions:    newMemorySessionStore(),
		ArgonParams: auth.DefaultArgon2Params,
	}
	handler := transport.Handler{Service: svc}

	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}

	padding := strings.Repeat("a", 256<<10)
	form := url.Values{
		"email":    {"oversize@example.com"},
		"password": {"Amber-River7!Cedar"},
		"padding":  {padding},
	}
	req := httptest.NewRequest(http.MethodPost, "/member/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(form.Encode()))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("POST /member/login accepted oversized form (%d padding bytes)", len(padding))
	}
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("POST /member/login status=%d, want 413", rr.Code)
	}
}

func TestAdminMemberMutationRoutesRequirePermissionAndCSRF(t *testing.T) {
	store := &trackedMemberStore{memoryMemberStore: newMemoryMemberStore()}
	store.members["member-1"] = &members.Member{ID: "member-1", Email: "member@example.test", Active: true}
	sessions := newMemorySessionStore()
	service := members.Service{Members: store, Sessions: sessions, ArgonParams: auth.DefaultArgon2Params}
	tokenA, sessionA, err := service.CreateSession(context.Background(), "member-1", "browser", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tokenB, sessionB, err := service.CreateSession(context.Background(), "member-1", "mobile", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	handler := transport.Handler{Service: service}
	mux := http.NewServeMux()
	registered := make(map[string]bool)
	for _, route := range handler.Routes() {
		registered[route.Pattern] = true
		mux.Handle(route.Pattern, route.Handler)
	}
	protected := security.CSRFMiddleware(false, nil)(mux)
	base := transport.AdminBasePath
	mutationRoutes := []struct {
		pattern string
		method  string
		path    string
	}{
		{pattern: "PATCH " + base + "/{id}", method: http.MethodPatch, path: base + "/member-1"},
		{pattern: "POST " + base + "/{id}/verify-email", method: http.MethodPost, path: base + "/member-1/verify-email"},
		{pattern: "POST " + base + "/{id}/resend-verification", method: http.MethodPost, path: base + "/member-1/resend-verification"},
		{pattern: "POST " + base + "/{id}/sessions/{session_id}/revoke", method: http.MethodPost, path: base + "/member-1/sessions/" + sessionA.ID + "/revoke"},
		{pattern: "POST " + base + "/{id}/sessions/revoke-all", method: http.MethodPost, path: base + "/member-1/sessions/revoke-all"},
	}
	deniedUsers := []struct {
		name string
		user *auth.User
	}{
		{name: "editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}},
		{name: "viewer", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}},
		{name: "admin members.manage deny", user: &auth.User{ID: "admin-deny", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: false}}},
		{name: "inactive admin", user: &auth.User{ID: "inactive", Role: auth.RoleAdmin, Active: false}},
	}
	for _, route := range mutationRoutes {
		if !registered[route.pattern] {
			t.Fatalf("mutation route %q is not registered", route.pattern)
		}
		t.Run(route.pattern, func(t *testing.T) {
			for _, tc := range deniedUsers {
				t.Run(tc.name, func(t *testing.T) {
					request := httptest.NewRequest(route.method, route.path, strings.NewReader(`{"active":false,"email_verified":true}`))
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set(security.CSRFHeaderName, "csrf-token")
					request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "csrf-token"})
					request = request.WithContext(auth.WithUser(request.Context(), tc.user))
					response := httptest.NewRecorder()
					protected.ServeHTTP(response, request)
					if response.Code != http.StatusForbidden {
						t.Fatalf("valid-CSRF %s request status=%d, want 403; body=%s", tc.name, response.Code, response.Body.String())
					}
				})
			}

			for _, csrfCase := range []struct {
				name   string
				body   string
				header string
				cookie string
			}{
				{name: "missing"},
				{name: "mismatched", body: `{"active":false,"email_verified":true}`, header: "wrong-token", cookie: "csrf-token"},
			} {
				t.Run("csrf "+csrfCase.name, func(t *testing.T) {
					request := httptest.NewRequest(route.method, route.path, strings.NewReader(csrfCase.body))
					request.Header.Set("Content-Type", "application/json")
					if csrfCase.header != "" {
						request.Header.Set(security.CSRFHeaderName, csrfCase.header)
					}
					if csrfCase.cookie != "" {
						request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: csrfCase.cookie})
					}
					request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
					response := httptest.NewRecorder()
					protected.ServeHTTP(response, request)
					if response.Code != http.StatusForbidden || !strings.Contains(strings.ToLower(response.Body.String()), "csrf") {
						t.Fatalf("%s CSRF request status=%d body=%q, want CSRF 403", csrfCase.name, response.Code, response.Body.String())
					}
				})
			}
		})
	}

	if store.activeWrites != 0 || store.verifiedWrites != 0 {
		t.Fatalf("denied requests reached member writes: active=%d email-verified=%d", store.activeWrites, store.verifiedWrites)
	}
	member := store.members["member-1"]
	if !member.Active || member.EmailVerifiedAt != nil {
		t.Fatalf("denied request changed member state: %+v", member)
	}
	for _, token := range []string{tokenA, tokenB} {
		if _, _, err := service.ValidateSession(context.Background(), token); err != nil {
			t.Fatalf("denied request revoked a member session: %v", err)
		}
	}
	if sessionA.ID == sessionB.ID {
		t.Fatal("test requires distinct member sessions")
	}
}
