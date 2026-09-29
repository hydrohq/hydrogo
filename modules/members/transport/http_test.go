package transport_test

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	memberstore "github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/members/transport"
	_ "modernc.org/sqlite"
)

type memoryMemberStore struct {
	members map[string]*members.Member
	byEmail map[string]*members.Member
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

func (m *memoryMemberStore) SetActive(_ context.Context, _ string, _ bool) error {
	return nil
}

func (m *memoryMemberStore) SetEmailVerified(_ context.Context, _ string, _ bool) error {
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

func (s *memorySessionStore) ListByMemberID(_ context.Context, _ string) ([]members.Session, error) {
	return nil, nil
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
