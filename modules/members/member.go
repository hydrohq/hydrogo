package members

import (
	"context"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

var (
	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberExists        = errors.New("member with this email already exists")
	ErrInvalidCredentials  = errors.New("invalid member credentials")
	ErrMemberInactive      = errors.New("member account is inactive")
	ErrMemberRepoRequired  = errors.New("member repository is required")
	ErrMemberSessionAbsent = errors.New("member session repository is required")
)

const SessionCookieName = "hydro_member_session"

type Member struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Active          bool       `json:"active"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Session struct {
	ID         string    `json:"id"`
	MemberID   string    `json:"member_id"`
	TokenHash  string    `json:"-"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
}

type AuditEvent struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	TargetID  string `json:"target_id"`
	RequestID string `json:"request_id,omitempty"`
}

type MemberFilter struct {
	Query  string
	Status string // "all", "active", "inactive", "verified", "unverified"
	Limit  int
	Offset int
}

type MemberListResult struct {
	Members    []Member `json:"members"`
	TotalCount int      `json:"total_count"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
}

type MemberStats struct {
	TotalMembers    int `json:"total_members"`
	ActiveMembers   int `json:"active_members"`
	VerifiedMembers int `json:"verified_members"`
}

type Repository interface {
	Create(context.Context, *Member) error
	GetByID(context.Context, string) (*Member, error)
	GetByEmail(context.Context, string) (*Member, error)
	List(context.Context, int) ([]Member, error)
	Search(context.Context, MemberFilter) (MemberListResult, error)
	SetActive(context.Context, string, bool) error
	SetEmailVerified(context.Context, string, bool) error
	UpdateLastLogin(context.Context, string, time.Time) error
	Stats(context.Context) (MemberStats, error)
}

type SessionRepository interface {
	Create(context.Context, *Session) error
	GetByTokenHash(context.Context, string) (*Session, error)
	DeleteByTokenHash(context.Context, string) error
	DeleteByID(context.Context, string) error
	DeleteByMemberID(context.Context, string) error
	ListByMemberID(context.Context, string) ([]Session, error)
	Touch(context.Context, string, time.Time) error
}

type AuditStore interface {
	ListByTargetID(context.Context, string, int) ([]AuditEvent, error)
}

type Service struct {
	Members     Repository
	Sessions    SessionRepository
	Audit       AuditStore
	ArgonParams auth.Argon2Params
}

func (s Service) Register(ctx context.Context, email, password string) (*Member, error) {
	if guards.IsNil(s.Members) {
		return nil, ErrMemberRepoRequired
	}
	email = auth.NormalizeEmail(email)
	if email == "" {
		return nil, auth.ErrInvalidCredentials
	}
	if err := auth.ValidatePasswordStrength(password); err != nil {
		return nil, err
	}
	if existing, err := s.Members.GetByEmail(ctx, email); err == nil && existing != nil {
		return nil, ErrMemberExists
	} else if err != nil && !errors.Is(err, ErrMemberNotFound) {
		return nil, err
	}
	hash, err := auth.HashPassword(password, s.ArgonParams)
	if err != nil {
		return nil, err
	}
	member := &Member{ID: newID(), Email: email, PasswordHash: hash, Active: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.Members.Create(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (s Service) Authenticate(ctx context.Context, email, password string) (*Member, error) {
	if guards.IsNil(s.Members) {
		return nil, ErrMemberRepoRequired
	}
	member, err := s.Members.GetByEmail(ctx, auth.NormalizeEmail(email))
	if err != nil || member == nil {
		return nil, ErrInvalidCredentials
	}
	if !member.Active {
		return nil, ErrMemberInactive
	}
	valid, err := auth.VerifyPassword(password, member.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}
	now := time.Now().UTC()
	_ = s.Members.UpdateLastLogin(ctx, member.ID, now)
	member.LastLoginAt = &now
	return member, nil
}

func (s Service) CreateSession(ctx context.Context, memberID, ua, ip string, ttl time.Duration) (string, *Session, error) {
	if guards.IsNil(s.Sessions) {
		return "", nil, ErrMemberSessionAbsent
	}
	raw, hash, err := auth.GenerateSessionToken()
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	session := &Session{ID: newID(), MemberID: memberID, TokenHash: hash, ExpiresAt: now.Add(ttl), CreatedAt: now, LastSeenAt: now, UserAgent: ua, IPAddress: ip}
	if err := s.Sessions.Create(ctx, session); err != nil {
		return "", nil, err
	}
	return raw, session, nil
}

func (s Service) ValidateSession(ctx context.Context, raw string) (*Session, *Member, error) {
	if guards.IsNil(s.Sessions) || guards.IsNil(s.Members) {
		return nil, nil, ErrMemberSessionAbsent
	}
	session, err := s.Sessions.GetByTokenHash(ctx, auth.HashSessionToken(raw))
	if err != nil || session == nil || !session.ExpiresAt.After(time.Now().UTC()) {
		return nil, nil, ErrInvalidCredentials
	}
	member, err := s.Members.GetByID(ctx, session.MemberID)
	if err != nil || member == nil || !member.Active {
		return nil, nil, ErrMemberInactive
	}
	return session, member, nil
}

func (s Service) Revoke(ctx context.Context, raw string) error {
	if guards.IsNil(s.Sessions) {
		return ErrMemberSessionAbsent
	}
	return s.Sessions.DeleteByTokenHash(ctx, auth.HashSessionToken(raw))
}

func (s Service) RevokeSessionByID(ctx context.Context, sessionID string) error {
	if guards.IsNil(s.Sessions) {
		return ErrMemberSessionAbsent
	}
	return s.Sessions.DeleteByID(ctx, sessionID)
}

func (s Service) RevokeAllSessions(ctx context.Context, memberID string) error {
	if guards.IsNil(s.Sessions) {
		return ErrMemberSessionAbsent
	}
	return s.Sessions.DeleteByMemberID(ctx, memberID)
}

func (s Service) Stats(ctx context.Context) (MemberStats, error) {
	if guards.IsNil(s.Members) {
		return MemberStats{}, ErrMemberRepoRequired
	}
	return s.Members.Stats(ctx)
}

func newID() string {
	raw, _, err := auth.GenerateSessionToken()
	if err != nil {
		return "member-id-unavailable"
	}
	return raw[:32]
}
