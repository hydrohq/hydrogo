package transport_test

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
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

	// 1. GET /member/register renders 200 and includes CSRF field
	req := httptest.NewRequest("GET", "/member/register", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /member/register status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `name="_csrf"`) {
		t.Fatalf("missing _csrf in register page body")
	}
	if !strings.Contains(rr.Body.String(), `data-password-strength`) || !strings.Contains(rr.Body.String(), `data-password-meter`) || !strings.Contains(rr.Body.String(), `data-password-feedback`) {
		t.Fatalf("registration page is missing accessible password guidance")
	}

	// 2. POST /member/register creates account and sets cookie
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

	// 3. GET /member/account with session cookie
	accReq := httptest.NewRequest("GET", "/member/account", nil)
	accReq.AddCookie(sessionCookie)
	accRR := httptest.NewRecorder()
	mux.ServeHTTP(accRR, accReq)
	if accRR.Code != http.StatusOK {
		t.Fatalf("GET /member/account status = %d", accRR.Code)
	}
	if !strings.Contains(accRR.Body.String(), "newuser@example.com") {
		t.Fatalf("account page did not display member email")
	}

	// 4. GET /member/me with session cookie returns 200 OK and email in JSON
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

	// 5. GET /member/me without session cookie returns 401 Unauthorized
	unauthReq := httptest.NewRequest("GET", "/member/me", nil)
	unauthRR := httptest.NewRecorder()
	mux.ServeHTTP(unauthRR, unauthReq)
	if unauthRR.Code != http.StatusUnauthorized {
		t.Fatalf("GET /member/me unauthenticated status = %d, expected 401", unauthRR.Code)
	}
}

func TestAdminMemberXLSXExportIsProtectedAndValid(t *testing.T) {
	store := fixedMemberStore{memoryMemberStore: newMemoryMemberStore(), rows: []members.Member{{ID: "m1", Email: "İstanbul@example.test", Active: true, CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}, {ID: "m2", Email: "=SUM(A1:A2)", CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)}}}
	svc := members.Service{Members: store, Sessions: newMemorySessionStore()}
	handler := transport.Handler{Service: svc}
	routes := map[string]http.Handler{}
	for _, route := range handler.Routes() {
		if strings.HasPrefix(route.Pattern, "GET /admin/members/export.") {
			routes[route.Pattern] = route.Handler
		}
	}
	export := routes["GET /admin/members/export.xlsx"]
	if export == nil || routes["GET /admin/members/export.csv"] == nil {
		t.Fatal("member spreadsheet export routes missing")
	}
	request := httptest.NewRequest(http.MethodGet, "/admin/members/export.xlsx", nil)
	response := httptest.NewRecorder()
	export.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unauthenticated export status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/admin/members/export.xlsx", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response = httptest.NewRecorder()
	export.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "spreadsheetml") {
		t.Fatalf("export status=%d content-type=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("invalid xlsx archive: %v", err)
	}
	if len(archive.File) < 5 {
		t.Fatalf("workbook has only %d parts", len(archive.File))
	}
	var sheet string
	for _, file := range archive.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(reader)
			_ = reader.Close()
			if err != nil {
				t.Fatal(err)
			}
			sheet = string(body)
		}
	}
	if !strings.Contains(sheet, "İstanbul@example.test") {
		t.Fatalf("workbook lost UTF-8 member email: %s", sheet)
	}
	csvRequest := httptest.NewRequest(http.MethodGet, "/admin/members/export.csv", nil)
	csvRequest = csvRequest.WithContext(auth.WithUser(csvRequest.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	csvResponse := httptest.NewRecorder()
	routes["GET /admin/members/export.csv"].ServeHTTP(csvResponse, csvRequest)
	if csvResponse.Code != http.StatusOK || !strings.Contains(csvResponse.Body.String(), "İstanbul@example.test") || !strings.Contains(csvResponse.Body.String(), "'=SUM(A1:A2)") {
		t.Fatalf("CSV export status=%d body=%q", csvResponse.Code, csvResponse.Body.String())
	}
}

func TestAdminSetActiveRevokesBrowserSessions(t *testing.T) {
	memberStore := newMemoryMemberStore()
	memberStore.members["member-1"] = &members.Member{ID: "member-1", Email: "member@example.com", Active: true}
	sessions := newMemorySessionStore()
	svc := members.Service{Members: memberStore, Sessions: sessions}
	token, session, err := svc.CreateSession(context.Background(), "member-1", "test", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handler := transport.Handler{Service: svc}
	request := httptest.NewRequest(http.MethodPost, "/admin/members/member-1/active", strings.NewReader("active=false"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if _, _, err := svc.ValidateSession(context.Background(), token); !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("deactivated member session still valid: session=%v err=%v", session.ID, err)
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
		if route.Pattern == "POST /admin/members/{id}/sessions/{session_id}/revoke" {
			revoke = route.Handler
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/members/member-a/sessions/"+sessionB.ID+"/revoke", nil)
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
	request = httptest.NewRequest(http.MethodPost, "/admin/members/member-a/sessions/"+sessionA.ID+"/revoke", nil)
	request.SetPathValue("id", "member-a")
	request.SetPathValue("session_id", sessionA.ID)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response = httptest.NewRecorder()
	revoke.ServeHTTP(response, request)
	if _, _, err := svc.ValidateSession(context.Background(), tokenA); !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("member A session survived own revoke: %v", err)
	}
	if response.Code != http.StatusSeeOther {
		t.Fatalf("own revoke status = %d, want %d", response.Code, http.StatusSeeOther)
	}
}

func TestAdminSetActiveRejectsExternalRefererRedirect(t *testing.T) {
	svc := members.Service{Members: newMemoryMemberStore(), Sessions: newMemorySessionStore()}
	handler := transport.Handler{Service: svc}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	req := httptest.NewRequest("POST", "/admin/members/member-1/active", strings.NewReader("active=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://evil.example/phish")
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "/admin/members/member-1" {
		t.Fatalf("Location = %q, want local fallback", got)
	}
}

func TestAdminMemberDetailEnforcesAccessAndHandlesMissingSessionStore(t *testing.T) {
	store := newMemoryMemberStore()
	store.members["m1"] = &members.Member{ID: "m1", Email: "member@example.com", Active: true}
	handler := transport.Handler{Service: members.Service{Members: store}}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	unauthorized := httptest.NewRecorder()
	mux.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/admin/members/m1", nil))
	if unauthorized.Code != http.StatusForbidden {
		t.Fatalf("unauthorized admin detail status = %d, want 403", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/admin/members/m1", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("admin detail with missing session store status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "member@example.com") {
		t.Fatal("authorized admin detail did not render member identity")
	}
}

func TestMemberListOffersVerificationResendOnlyForUnverifiedAccounts(t *testing.T) {
	rows := []members.Member{{ID: "pending", Email: "pending@example.test", Active: true}, {ID: "verified", Email: "verified@example.test", Active: true, EmailVerifiedAt: func() *time.Time { now := time.Now(); return &now }()}}
	store := fixedMemberStore{memoryMemberStore: newMemoryMemberStore(), rows: rows}
	handler := transport.Handler{Service: members.Service{Members: store}, VerificationEnabled: true}
	request := httptest.NewRequest(http.MethodGet, "/admin/members", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}))
	response := httptest.NewRecorder()
	for _, route := range handler.Routes() {
		if route.Pattern == "GET /admin/members" {
			route.Handler.ServeHTTP(response, request)
			break
		}
	}
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "/admin/members/pending/resend-verification") || strings.Contains(body, "/admin/members/verified/resend-verification") {
		t.Fatalf("member list status=%d, unexpected resend controls in body", response.Code)
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
		if route.Pattern == "POST /admin/members/{id}/resend-verification" {
			resend = route.Handler
		}
		if route.Pattern == "GET /member/email/verify" {
			verify = route.Handler
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/members/member-1/resend-verification", nil)
	request.SetPathValue("id", "member-1")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	resend.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || notifier.email != "member@example.test" || !strings.HasPrefix(notifier.link, "https://hydro.example.test/member/email/verify?token=") {
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

func TestAdminMemberReadRoutesUseMemberManageOverrides(t *testing.T) {
	store := newMemoryMemberStore()
	store.members["member-1"] = &members.Member{ID: "member-1", Email: "member@example.test", Active: true}
	handler := transport.Handler{Service: members.Service{Members: store}}
	mux := http.NewServeMux()
	registered := make(map[string]bool)
	for _, route := range handler.Routes() {
		registered[route.Pattern] = true
		mux.Handle(route.Pattern, route.Handler)
	}
	routes := []struct{ pattern, path string }{
		{pattern: "GET /admin/members", path: "/admin/members"},
		{pattern: "GET /admin/members/export.xlsx", path: "/admin/members/export.xlsx"},
		{pattern: "GET /admin/members/export.csv", path: "/admin/members/export.csv"},
		{pattern: "GET /admin/members/{id}", path: "/admin/members/member-1"},
	}
	for _, tc := range []struct {
		name string
		user *auth.User
		want int
	}{
		{name: "editor default denied", user: &auth.User{Role: auth.RoleEditor, Active: true}, want: http.StatusForbidden},
		{name: "editor allow override", user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: true}}, want: http.StatusOK},
		{name: "viewer allow override", user: &auth.User{Role: auth.RoleViewer, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: true}}, want: http.StatusOK},
		{name: "admin deny override", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: false}}, want: http.StatusForbidden},
		{name: "inactive grant denied", user: &auth.User{Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionMembersManage: true}}, want: http.StatusForbidden},
		{name: "admin default", user: &auth.User{Role: auth.RoleAdmin, Active: true}, want: http.StatusOK},
	} {
		for _, route := range routes {
			if !registered[route.pattern] {
				t.Fatalf("read route %q is not registered", route.pattern)
			}
			t.Run(route.pattern+"/"+tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, route.path, nil)
				req = req.WithContext(auth.WithUser(req.Context(), tc.user))
				res := httptest.NewRecorder()
				mux.ServeHTTP(res, req)
				if res.Code != tc.want {
					t.Fatalf("GET %s status=%d want=%d body=%s", route.path, res.Code, tc.want, res.Body.String())
				}
			})
		}
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
	mutationRoutes := []struct {
		pattern string
		path    string
	}{
		{pattern: "POST /admin/members/{id}", path: "/admin/members/member-1"},
		{pattern: "POST /admin/members/{id}/active", path: "/admin/members/member-1/active"},
		{pattern: "POST /admin/members/{id}/verify-email", path: "/admin/members/member-1/verify-email"},
		{pattern: "POST /admin/members/{id}/resend-verification", path: "/admin/members/member-1/resend-verification"},
		{pattern: "POST /admin/members/{id}/sessions/{session_id}/revoke", path: "/admin/members/member-1/sessions/" + sessionA.ID + "/revoke"},
		{pattern: "POST /admin/members/{id}/sessions/revoke-all", path: "/admin/members/member-1/sessions/revoke-all"},
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
					form := url.Values{"_csrf": {"csrf-token"}, "active": {"false"}, "verified": {"true"}}
					request := httptest.NewRequest(http.MethodPost, route.path, strings.NewReader(form.Encode()))
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
				{name: "mismatched", body: "_csrf=wrong-token&active=false&verified=true", header: "wrong-token", cookie: "csrf-token"},
			} {
				t.Run("csrf "+csrfCase.name, func(t *testing.T) {
					request := httptest.NewRequest(http.MethodPost, route.path, strings.NewReader(csrfCase.body))
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
