package members

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/auth"
	"github.com/hydrohq/hydrogo/internal/guards"
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
	GetByID(context.Context, string, string) (*Session, error)
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

func (s Service) RevokeSessionByID(ctx context.Context, memberID, sessionID string) error {
	if guards.IsNil(s.Sessions) {
		return ErrMemberSessionAbsent
	}
	if _, err := s.Sessions.GetByID(ctx, memberID, sessionID); err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return ErrMemberNotFound
		}
		return err
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

type EmailUpdater interface {
	UpdateEmail(ctx context.Context, id, email string) error
}

type PasswordUpdater interface {
	UpdatePasswordHash(ctx context.Context, id, passwordHash string) error
}

type MemberDeleter interface {
	Delete(ctx context.Context, id string) error
}

func (s Service) UpdateEmail(ctx context.Context, memberID, newEmail string) error {
	if guards.IsNil(s.Members) {
		return ErrMemberRepoRequired
	}
	updater, ok := s.Members.(EmailUpdater)
	if !ok {
		return ErrMemberRepoRequired
	}
	newEmail = auth.NormalizeEmail(newEmail)
	if newEmail == "" || !strings.Contains(newEmail, "@") {
		return auth.ErrInvalidCredentials
	}
	if existing, err := s.Members.GetByEmail(ctx, newEmail); err == nil && existing != nil && existing.ID != memberID {
		return ErrMemberExists
	} else if err != nil && !errors.Is(err, ErrMemberNotFound) {
		return err
	}
	return updater.UpdateEmail(ctx, memberID, newEmail)
}

// UpdatePassword is the admin-reset path: it revokes every session of the member.
func (s Service) UpdatePassword(ctx context.Context, memberID, newPassword string) error {
	if err := s.setPassword(ctx, memberID, newPassword); err != nil {
		return err
	}
	if !guards.IsNil(s.Sessions) {
		return s.Sessions.DeleteByMemberID(ctx, memberID)
	}
	return nil
}

func (s Service) setPassword(ctx context.Context, memberID, newPassword string) error {
	if guards.IsNil(s.Members) {
		return ErrMemberRepoRequired
	}
	updater, ok := s.Members.(PasswordUpdater)
	if !ok {
		return ErrMemberRepoRequired
	}
	runes := []rune(strings.TrimSpace(newPassword))
	if len(runes) < 6 || len(runes) > 1024 {
		return auth.ErrWeakPassword
	}
	hash, err := auth.HashPassword(newPassword, s.ArgonParams)
	if err != nil {
		return err
	}
	return updater.UpdatePasswordHash(ctx, memberID, hash)
}

// ChangeOwnPassword revokes every session except keepSessionID (the caller's).
func (s Service) ChangeOwnPassword(ctx context.Context, memberID, keepSessionID, currentPassword, newPassword string) error {
	if guards.IsNil(s.Members) {
		return ErrMemberRepoRequired
	}
	member, err := s.Members.GetByID(ctx, memberID)
	if err != nil || member == nil {
		return ErrMemberNotFound
	}
	valid, err := auth.VerifyPassword(currentPassword, member.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidCredentials
	}
	if err := s.setPassword(ctx, memberID, newPassword); err != nil {
		return err
	}
	if guards.IsNil(s.Sessions) {
		return nil
	}
	list, err := s.Sessions.ListByMemberID(ctx, memberID)
	if err != nil {
		return err
	}
	for _, sess := range list {
		if sess.ID != keepSessionID {
			if err := s.Sessions.DeleteByID(ctx, sess.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s Service) ChangeOwnEmail(ctx context.Context, memberID, currentPassword, newEmail string) error {
	if guards.IsNil(s.Members) {
		return ErrMemberRepoRequired
	}
	member, err := s.Members.GetByID(ctx, memberID)
	if err != nil || member == nil {
		return ErrMemberNotFound
	}
	valid, err := auth.VerifyPassword(currentPassword, member.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidCredentials
	}
	if err := s.UpdateEmail(ctx, memberID, newEmail); err != nil {
		return err
	}
	// The new address is unproven: drop the verified flag.
	return s.Members.SetEmailVerified(ctx, memberID, false)
}

func (s Service) DeleteMember(ctx context.Context, memberID string) error {
	if guards.IsNil(s.Members) {
		return ErrMemberRepoRequired
	}
	deleter, ok := s.Members.(MemberDeleter)
	if !ok {
		return ErrMemberRepoRequired
	}
	if !guards.IsNil(s.Sessions) {
		_ = s.Sessions.DeleteByMemberID(ctx, memberID)
	}
	return deleter.Delete(ctx, memberID)
}

func (s Service) AdminCreateMember(ctx context.Context, email, password string, active, verified bool) (*Member, error) {
	if guards.IsNil(s.Members) {
		return nil, ErrMemberRepoRequired
	}
	email = auth.NormalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, auth.ErrInvalidCredentials
	}
	runes := []rune(strings.TrimSpace(password))
	if len(runes) < 6 || len(runes) > 1024 {
		return nil, auth.ErrWeakPassword
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
	now := time.Now().UTC()
	var verifiedAt *time.Time
	if verified {
		verifiedAt = &now
	}
	member := &Member{
		ID:              newID(),
		Email:           email,
		PasswordHash:    hash,
		Active:          active,
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.Members.Create(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

func newID() string {
	raw, _, err := auth.GenerateSessionToken()
	if err != nil {
		return "member-id-unavailable"
	}
	return raw[:32]
}
