package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type Store struct {
	DB      *sql.DB
	Dialect database.Dialect
}

type SessionStore Store

func (s SessionStore) ph(n int) string { return Store(s).ph(n) }

func (s Store) ph(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s Store) Create(ctx context.Context, m *members.Member) error {
	var verifiedAt, lastLoginAt any
	var createdAtVal any = m.CreatedAt
	var updatedAtVal any = m.UpdatedAt
	if m.EmailVerifiedAt != nil {
		if s.Dialect == database.SQLite {
			verifiedAt = m.EmailVerifiedAt.Format(time.RFC3339Nano)
		} else {
			verifiedAt = m.EmailVerifiedAt
		}
	}
	if m.LastLoginAt != nil {
		if s.Dialect == database.SQLite {
			lastLoginAt = m.LastLoginAt.Format(time.RFC3339Nano)
		} else {
			lastLoginAt = m.LastLoginAt
		}
	}
	if s.Dialect == database.SQLite {
		createdAtVal = m.CreatedAt.Format(time.RFC3339Nano)
		updatedAtVal = m.UpdatedAt.Format(time.RFC3339Nano)
	}
	_, err := s.DB.ExecContext(ctx, "INSERT INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES ("+s.ph(1)+","+s.ph(2)+","+s.ph(3)+","+s.ph(4)+","+s.ph(5)+","+s.ph(6)+","+s.ph(7)+","+s.ph(8)+")", m.ID, m.Email, m.PasswordHash, m.Active, verifiedAt, lastLoginAt, createdAtVal, updatedAtVal)
	return mapMemberCreateError(err)
}

// mapMemberCreateError keeps driver-specific uniqueness failures behind the
// repository contract. The service can therefore return the same safe domain
// error for SQLite and PostgreSQL, including races between pre-check and
// insert.
func mapMemberCreateError(err error) error {
	if err == nil {
		return nil
	}
	if database.IsUniqueViolation(err) {
		return members.ErrMemberExists
	}
	return err
}

func (s Store) GetByID(ctx context.Context, id string) (*members.Member, error) {
	return s.get(ctx, "id", id)
}

func (s Store) GetByEmail(ctx context.Context, email string) (*members.Member, error) {
	return s.get(ctx, "email", auth.NormalizeEmail(email))
}

func (s Store) get(ctx context.Context, field, value string) (*members.Member, error) {
	var m members.Member
	var active any
	var verified, lastLogin, created, updated any
	q := "SELECT id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at FROM members WHERE " + field + " = " + s.ph(1)
	err := s.DB.QueryRowContext(ctx, q, value).Scan(&m.ID, &m.Email, &m.PasswordHash, &active, &verified, &lastLogin, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, members.ErrMemberNotFound
	}
	if err != nil {
		return nil, err
	}
	m.Active = parseBool(active)
	m.EmailVerifiedAt = parseTimePtr(verified)
	m.LastLoginAt = parseTimePtr(lastLogin)
	m.CreatedAt = parseTime(created)
	m.UpdatedAt = parseTime(updated)
	return &m, nil
}

func (s Store) List(ctx context.Context, limit int) ([]members.Member, error) {
	res, err := s.Search(ctx, members.MemberFilter{Limit: limit})
	if err != nil {
		return nil, err
	}
	return res.Members, nil
}

func (s Store) Search(ctx context.Context, filter members.MemberFilter) (members.MemberListResult, error) {
	if filter.Limit < 1 || filter.Limit > 500 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if q := strings.TrimSpace(filter.Query); q != "" {
		whereClauses = append(whereClauses, "(email LIKE "+s.ph(argIdx)+")")
		args = append(args, "%"+auth.NormalizeEmail(q)+"%")
		argIdx++
	}

	switch filter.Status {
	case "active":
		whereClauses = append(whereClauses, "active = "+s.ph(argIdx))
		args = append(args, true)
		argIdx++
	case "inactive":
		whereClauses = append(whereClauses, "active = "+s.ph(argIdx))
		args = append(args, false)
		argIdx++
	case "verified":
		whereClauses = append(whereClauses, "email_verified_at IS NOT NULL")
	case "unverified":
		whereClauses = append(whereClauses, "email_verified_at IS NULL")
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM members WHERE " + whereSQL
	if err := s.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return members.MemberListResult{}, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, filter.Limit, filter.Offset)
	query := fmt.Sprintf("SELECT id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at FROM members WHERE %s ORDER BY created_at DESC LIMIT %s OFFSET %s",
		whereSQL, s.ph(argIdx), s.ph(argIdx+1))

	rows, err := s.DB.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return members.MemberListResult{}, err
	}
	defer rows.Close()

	result := []members.Member{}
	for rows.Next() {
		var m members.Member
		var active any
		var verified, lastLogin, created, updated any
		if err := rows.Scan(&m.ID, &m.Email, &m.PasswordHash, &active, &verified, &lastLogin, &created, &updated); err != nil {
			return members.MemberListResult{}, err
		}
		m.PasswordHash = ""
		m.Active = parseBool(active)
		m.EmailVerifiedAt = parseTimePtr(verified)
		m.LastLoginAt = parseTimePtr(lastLogin)
		m.CreatedAt = parseTime(created)
		m.UpdatedAt = parseTime(updated)
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return members.MemberListResult{}, err
	}

	return members.MemberListResult{
		Members:    result,
		TotalCount: total,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}, nil
}

func (s Store) SetActive(ctx context.Context, id string, active bool) error {
	result, err := s.DB.ExecContext(ctx, "UPDATE members SET active = "+s.ph(1)+", updated_at = "+s.ph(2)+" WHERE id = "+s.ph(3), active, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return members.ErrMemberNotFound
	}
	return nil
}

func (s Store) SetEmailVerified(ctx context.Context, id string, verified bool) error {
	var verifiedAt any
	if verified {
		verifiedAt = time.Now().UTC()
	}
	result, err := s.DB.ExecContext(ctx, "UPDATE members SET email_verified_at = "+s.ph(1)+", updated_at = "+s.ph(2)+" WHERE id = "+s.ph(3), verifiedAt, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return members.ErrMemberNotFound
	}
	return nil
}

func (s Store) UpdateLastLogin(ctx context.Context, id string, t time.Time) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE members SET last_login_at = "+s.ph(1)+" WHERE id = "+s.ph(2), t, id)
	return err
}

func (s Store) Stats(ctx context.Context) (members.MemberStats, error) {
	var st members.MemberStats
	activeCondition := "active = 1"
	if s.Dialect == database.Postgres {
		activeCondition = "active = TRUE"
	}
	row := s.DB.QueryRowContext(ctx, "SELECT COUNT(*), COALESCE(SUM(CASE WHEN "+activeCondition+" THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN email_verified_at IS NOT NULL THEN 1 ELSE 0 END), 0) FROM members")
	err := row.Scan(&st.TotalMembers, &st.ActiveMembers, &st.VerifiedMembers)
	return st, err
}

func (s SessionStore) Create(ctx context.Context, session *members.Session) error {
	var expiresAtVal any = session.ExpiresAt
	var createdAtVal any = session.CreatedAt
	var lastSeenAtVal any = session.LastSeenAt
	if s.Dialect == database.SQLite {
		expiresAtVal = session.ExpiresAt.Format(time.RFC3339Nano)
		createdAtVal = session.CreatedAt.Format(time.RFC3339Nano)
		lastSeenAtVal = session.LastSeenAt.Format(time.RFC3339Nano)
	}
	_, err := s.DB.ExecContext(ctx, "INSERT INTO member_sessions (id,member_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address) VALUES ("+s.ph(1)+","+s.ph(2)+","+s.ph(3)+","+s.ph(4)+","+s.ph(5)+","+s.ph(6)+","+s.ph(7)+","+s.ph(8)+")", session.ID, session.MemberID, session.TokenHash, expiresAtVal, createdAtVal, lastSeenAtVal, session.UserAgent, session.IPAddress)
	return err
}

func (s SessionStore) GetByTokenHash(ctx context.Context, hash string) (*members.Session, error) {
	var v members.Session
	var expires, created, lastSeen any
	err := s.DB.QueryRowContext(ctx, "SELECT id,member_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address FROM member_sessions WHERE token_hash = "+s.ph(1), hash).Scan(&v.ID, &v.MemberID, &v.TokenHash, &expires, &created, &lastSeen, &v.UserAgent, &v.IPAddress)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, members.ErrMemberNotFound
	}
	if err != nil {
		return nil, err
	}
	v.ExpiresAt = parseTime(expires)
	v.CreatedAt = parseTime(created)
	v.LastSeenAt = parseTime(lastSeen)
	return &v, nil
}

func (s SessionStore) DeleteByTokenHash(ctx context.Context, hash string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM member_sessions WHERE token_hash = "+s.ph(1), hash)
	return err
}

func (s SessionStore) DeleteByID(ctx context.Context, sessionID string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM member_sessions WHERE id = "+s.ph(1), sessionID)
	return err
}

func (s SessionStore) DeleteByMemberID(ctx context.Context, memberID string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM member_sessions WHERE member_id = "+s.ph(1), memberID)
	return err
}

func (s SessionStore) ListByMemberID(ctx context.Context, memberID string) ([]members.Session, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,member_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address FROM member_sessions WHERE member_id = "+s.ph(1)+" ORDER BY last_seen_at DESC LIMIT 100", memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []members.Session
	for rows.Next() {
		var v members.Session
		var expires, created, lastSeen any
		if err := rows.Scan(&v.ID, &v.MemberID, &v.TokenHash, &expires, &created, &lastSeen, &v.UserAgent, &v.IPAddress); err != nil {
			return nil, err
		}
		v.ExpiresAt = parseTime(expires)
		v.CreatedAt = parseTime(created)
		v.LastSeenAt = parseTime(lastSeen)
		result = append(result, v)
	}
	return result, rows.Err()
}

func (s SessionStore) Touch(ctx context.Context, hash string, t time.Time) error {
	var touchVal any = t
	if s.Dialect == database.SQLite {
		touchVal = t.Format(time.RFC3339Nano)
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE member_sessions SET last_seen_at = "+s.ph(1)+" WHERE token_hash = "+s.ph(2), touchVal, hash)
	return err
}

type AuditStore struct {
	DB      *sql.DB
	Dialect database.Dialect
}

func (s AuditStore) ListByTargetID(ctx context.Context, targetID string, limit int) ([]members.AuditEvent, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	ph := "?"
	if s.Dialect == database.Postgres {
		ph = "$1"
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT id, action, target_id, COALESCE(request_id, '') FROM audit_events WHERE target_id = "+ph+" ORDER BY id DESC LIMIT "+fmt.Sprint(limit), targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []members.AuditEvent
	for rows.Next() {
		var ev members.AuditEvent
		if err := rows.Scan(&ev.ID, &ev.Action, &ev.TargetID, &ev.RequestID); err != nil {
			return nil, err
		}
		result = append(result, ev)
	}
	return result, rows.Err()
}

func parseBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case int64:
		return x == 1
	case []byte:
		return string(x) == "1" || string(x) == "true"
	case string:
		return x == "1" || x == "true"
	}
	return false
}

func parseTime(v any) time.Time {
	if t := parseTimePtr(v); t != nil {
		return *t
	}
	return time.Time{}
}

func parseTimePtr(v any) *time.Time {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case time.Time:
		t := x.UTC()
		return &t
	case string:
		return parseTimeString(x)
	case []byte:
		return parseTimeString(string(x))
	}
	return nil
}

func parseTimeString(str string) *time.Time {
	str = strings.TrimSpace(str)
	if str == "" {
		return nil
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, str); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}
