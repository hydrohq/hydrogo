package members_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/modules/members"
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

func (m *memoryMemberStore) List(_ context.Context, limit int) ([]members.Member, error) {
	var list []members.Member
	for _, v := range m.members {
		list = append(list, *v)
	}
	return list, nil
}

func (m *memoryMemberStore) Search(_ context.Context, filter members.MemberFilter) (members.MemberListResult, error) {
	var list []members.Member
	for _, v := range m.members {
		list = append(list, *v)
	}
	return members.MemberListResult{Members: list, TotalCount: len(list)}, nil
}

func (m *memoryMemberStore) SetActive(_ context.Context, id string, active bool) error {
	if mem, ok := m.members[id]; ok {
		mem.Active = active
		return nil
	}
	return members.ErrMemberNotFound
}

func (m *memoryMemberStore) SetEmailVerified(_ context.Context, id string, verified bool) error {
	if mem, ok := m.members[id]; ok {
		if verified {
			now := time.Now().UTC()
			mem.EmailVerifiedAt = &now
		} else {
			mem.EmailVerifiedAt = nil
		}
		return nil
	}
	return members.ErrMemberNotFound
}

func (m *memoryMemberStore) UpdateLastLogin(_ context.Context, id string, t time.Time) error {
	if mem, ok := m.members[id]; ok {
		mem.LastLoginAt = &t
		return nil
	}
	return members.ErrMemberNotFound
}

func (m *memoryMemberStore) Stats(_ context.Context) (members.MemberStats, error) {
	return members.MemberStats{TotalMembers: len(m.members)}, nil
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

func (s *memorySessionStore) GetByID(_ context.Context, memberID, sessionID string) (*members.Session, error) {
	for _, session := range s.sessions {
		if session.MemberID == memberID && session.ID == sessionID {
			return session, nil
		}
	}
	return nil, members.ErrMemberNotFound
}

func (s *memorySessionStore) DeleteByMemberID(_ context.Context, memberID string) error {
	for k, v := range s.sessions {
		if v.MemberID == memberID {
			delete(s.sessions, k)
		}
	}
	return nil
}

func (s *memorySessionStore) ListByMemberID(_ context.Context, memberID string) ([]members.Session, error) {
	var list []members.Session
	for _, v := range s.sessions {
		if v.MemberID == memberID {
			list = append(list, *v)
		}
	}
	return list, nil
}

func (s *memorySessionStore) Touch(_ context.Context, hash string, t time.Time) error {
	if sess, ok := s.sessions[hash]; ok {
		sess.LastSeenAt = t
		return nil
	}
	return nil
}

func TestMemberRegistrationAndAuthentication(t *testing.T) {
	svc := members.Service{
		Members:     newMemoryMemberStore(),
		Sessions:    newMemorySessionStore(),
		ArgonParams: auth.DefaultArgon2Params,
	}

	ctx := context.Background()

	// 1. Password too short (< 8 chars)
	_, err := svc.Register(ctx, "user@example.com", "short")
	if !errors.Is(err, auth.ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword for short password, got: %v", err)
	}

	// 2. Successful Registration
	mem, err := svc.Register(ctx, "User@Example.com ", "Amber-River7!Cedar")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	if mem.Email != "user@example.com" {
		t.Fatalf("expected normalized email user@example.com, got: %s", mem.Email)
	}
	if !mem.Active {
		t.Fatal("expected new member to be active")
	}

	// 3. Duplicate email registration
	_, err = svc.Register(ctx, "user@example.com", "Amber-River7!Cedar")
	if !errors.Is(err, members.ErrMemberExists) {
		t.Fatalf("expected ErrMemberExists, got: %v", err)
	}

	// 4. Authenticate wrong password
	_, err = svc.Authenticate(ctx, "user@example.com", "wrongpass")
	if !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}

	// 5. Authenticate success
	authedMem, err := svc.Authenticate(ctx, "user@example.com", "Amber-River7!Cedar")
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}
	if authedMem.ID != mem.ID {
		t.Fatalf("expected authed member ID %s, got %s", mem.ID, authedMem.ID)
	}

	// 6. Session Creation & Validation
	rawToken, sess, err := svc.CreateSession(ctx, authedMem.ID, "test-agent", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("session creation failed: %v", err)
	}
	if rawToken == "" || sess == nil {
		t.Fatal("empty raw session token")
	}

	valSess, valMem, err := svc.ValidateSession(ctx, rawToken)
	if err != nil {
		t.Fatalf("validate session failed: %v", err)
	}
	if valMem.ID != mem.ID || valSess.ID != sess.ID {
		t.Fatalf("session validation mismatch")
	}

	// 7. Revoke session
	if err := svc.Revoke(ctx, rawToken); err != nil {
		t.Fatalf("revoke session failed: %v", err)
	}
	_, _, err = svc.ValidateSession(ctx, rawToken)
	if !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials after revoke, got: %v", err)
	}
}
