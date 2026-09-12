package transport_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	"github.com/ardaozyildirim/hydrogo/modules/members/transport"
)

type memoryMemberStore struct {
	members map[string]*members.Member
	byEmail map[string]*members.Member
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

	// 2. POST /member/register creates account and sets cookie
	form := url.Values{"email": {"newuser@example.com"}, "password": {"secret1234"}}
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
