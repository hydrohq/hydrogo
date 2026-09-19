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
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type Store struct {
	DB                 *sql.DB
	Dialect            database.Dialect
	RequireTenantScope bool
}

type SessionStore Store

func (s Store) within(ctx context.Context, fn func(database.Executor) error) error {
	if s.DB == nil {
		return database.ErrDatabaseRequired
	}
	return database.NewTenantTransactor(s.DB, s.Dialect, s.RequireTenantScope).Within(ctx, fn)
}

func (s Store) withinQuery(ctx context.Context, fn func(database.QueryExecutor) error) error {
	if s.DB == nil {
		return database.ErrDatabaseRequired
	}
	return database.NewTenantTransactor(s.DB, s.Dialect, s.RequireTenantScope).WithinQuery(ctx, fn)
}

func (s Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var result sql.Result
	err := s.within(ctx, func(exec database.Executor) error {
		var err error
		result, err = exec.ExecContext(ctx, query, args...)
		return err
	})
	return result, err
}

func (s Store) scanRow(ctx context.Context, query string, args []any, scan func(*sql.Row) error) error {
	return s.within(ctx, func(exec database.Executor) error {
		return scan(exec.QueryRowContext(ctx, query, args...))
	})
}

func (s SessionStore) ph(n int) string { return Store(s).ph(n) }

func (s SessionStore) tenantClause(ctx context.Context, position int) (string, any, error) {
	return Store(s).tenantClause(ctx, position)
}

func (s Store) ph(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s Store) tenantClause(ctx context.Context, position int) (string, any, error) {
	if !s.RequireTenantScope {
		return "", nil, nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", nil, err
	}
	return " AND tenant_id = " + s.ph(position), scope.TenantID, nil
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
	args := []any{m.ID, m.Email, m.PasswordHash, m.Active, verifiedAt, lastLoginAt, createdAtVal, updatedAtVal}
	query := "INSERT INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES (" + s.ph(1) + "," + s.ph(2) + "," + s.ph(3) + "," + s.ph(4) + "," + s.ph(5) + "," + s.ph(6) + "," + s.ph(7) + "," + s.ph(8) + ")"
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		query = "INSERT INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at,tenant_id) VALUES (" + s.ph(1) + "," + s.ph(2) + "," + s.ph(3) + "," + s.ph(4) + "," + s.ph(5) + "," + s.ph(6) + "," + s.ph(7) + "," + s.ph(8) + "," + s.ph(9) + ")"
		args = append(args, scope.TenantID)
	}
	_, err := s.exec(ctx, query, args...)
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
	clause, scopeID, err := s.tenantClause(ctx, 2)
	if err != nil {
		return nil, err
	}
	q := "SELECT id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at FROM members WHERE " + field + " = " + s.ph(1) + clause
	args := []any{value}
	if clause != "" {
		args = append(args, scopeID)
	}
	err = s.scanRow(ctx, q, args, func(row *sql.Row) error {
		return row.Scan(&m.ID, &m.Email, &m.PasswordHash, &active, &verified, &lastLogin, &created, &updated)
	})
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
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return members.MemberListResult{}, scopeErr
		}
		whereClauses = append(whereClauses, "tenant_id = "+s.ph(argIdx))
		args = append(args, scope.TenantID)
		argIdx++
	}

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
	if err := s.scanRow(ctx, countQuery, args, func(row *sql.Row) error { return row.Scan(&total) }); err != nil {
		return members.MemberListResult{}, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, filter.Limit, filter.Offset)
	query := fmt.Sprintf("SELECT id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at FROM members WHERE %s ORDER BY created_at DESC LIMIT %s OFFSET %s",
		whereSQL, s.ph(argIdx), s.ph(argIdx+1))

	result := []members.Member{}
	if err := s.withinQuery(ctx, func(exec database.QueryExecutor) error {
		rows, err := exec.QueryContext(ctx, query, listArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m members.Member
			var active any
			var verified, lastLogin, created, updated any
			if err := rows.Scan(&m.ID, &m.Email, &m.PasswordHash, &active, &verified, &lastLogin, &created, &updated); err != nil {
				return err
			}
			m.PasswordHash = ""
			m.Active = parseBool(active)
			m.EmailVerifiedAt = parseTimePtr(verified)
			m.LastLoginAt = parseTimePtr(lastLogin)
			m.CreatedAt = parseTime(created)
			m.UpdatedAt = parseTime(updated)
			result = append(result, m)
		}
		return rows.Err()
	}); err != nil {
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
	clause, scopeID, err := s.tenantClause(ctx, 4)
	if err != nil {
		return err
	}
	args := []any{active, time.Now().UTC(), id}
	if clause != "" {
		args = append(args, scopeID)
	}
	result, err := s.exec(ctx, "UPDATE members SET active = "+s.ph(1)+", updated_at = "+s.ph(2)+" WHERE id = "+s.ph(3)+clause, args...)
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
	clause, scopeID, err := s.tenantClause(ctx, 4)
	if err != nil {
		return err
	}
	args := []any{verifiedAt, time.Now().UTC(), id}
	if clause != "" {
		args = append(args, scopeID)
	}
	result, err := s.exec(ctx, "UPDATE members SET email_verified_at = "+s.ph(1)+", updated_at = "+s.ph(2)+" WHERE id = "+s.ph(3)+clause, args...)
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
	clause, scopeID, err := s.tenantClause(ctx, 3)
	if err != nil {
		return err
	}
	args := []any{t, id}
	if clause != "" {
		args = append(args, scopeID)
	}
	_, err = s.exec(ctx, "UPDATE members SET last_login_at = "+s.ph(1)+" WHERE id = "+s.ph(2)+clause, args...)
	return err
}

func (s Store) Stats(ctx context.Context) (members.MemberStats, error) {
	var st members.MemberStats
	activeCondition := "active = 1"
	if s.Dialect == database.Postgres {
		activeCondition = "active = TRUE"
	}
	clause, scopeID, err := s.tenantClause(ctx, 1)
	if err != nil {
		return st, err
	}
	args := []any{}
	if clause != "" {
		args = append(args, scopeID)
	}
	scanErr := s.scanRow(ctx, "SELECT COUNT(*), COALESCE(SUM(CASE WHEN "+activeCondition+" THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN email_verified_at IS NOT NULL THEN 1 ELSE 0 END), 0) FROM members WHERE 1=1"+clause, args, func(row *sql.Row) error {
		return row.Scan(&st.TotalMembers, &st.ActiveMembers, &st.VerifiedMembers)
	})
	return st, scanErr
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
	args := []any{session.ID, session.MemberID, session.TokenHash, expiresAtVal, createdAtVal, lastSeenAtVal, session.UserAgent, session.IPAddress}
	query := "INSERT INTO member_sessions (id,member_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address) VALUES (" + s.ph(1) + "," + s.ph(2) + "," + s.ph(3) + "," + s.ph(4) + "," + s.ph(5) + "," + s.ph(6) + "," + s.ph(7) + "," + s.ph(8) + ")"
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		query = "INSERT INTO member_sessions (id,member_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address,tenant_id) VALUES (" + s.ph(1) + "," + s.ph(2) + "," + s.ph(3) + "," + s.ph(4) + "," + s.ph(5) + "," + s.ph(6) + "," + s.ph(7) + "," + s.ph(8) + "," + s.ph(9) + ")"
		args = append(args, scope.TenantID)
	}
	_, err := Store(s).exec(ctx, query, args...)
	return err
}

func (s SessionStore) GetByTokenHash(ctx context.Context, hash string) (*members.Session, error) {
	var v members.Session
	var expires, created, lastSeen any
	clause, scopeID, err := s.tenantClause(ctx, 2)
	if err != nil {
		return nil, err
	}
	args := []any{hash}
	if clause != "" {
		args = append(args, scopeID)
	}
	err = Store(s).scanRow(ctx, "SELECT id,member_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address FROM member_sessions WHERE token_hash = "+s.ph(1)+clause, args, func(row *sql.Row) error {
		return row.Scan(&v.ID, &v.MemberID, &v.TokenHash, &expires, &created, &lastSeen, &v.UserAgent, &v.IPAddress)
	})
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
	clause, scopeID, err := s.tenantClause(ctx, 2)
	if err != nil {
		return err
	}
	args := []any{hash}
	if clause != "" {
		args = append(args, scopeID)
	}
	_, err = Store(s).exec(ctx, "DELETE FROM member_sessions WHERE token_hash = "+s.ph(1)+clause, args...)
	return err
}

func (s SessionStore) DeleteByID(ctx context.Context, sessionID string) error {
	clause, scopeID, err := s.tenantClause(ctx, 2)
	if err != nil {
		return err
	}
	args := []any{sessionID}
	if clause != "" {
		args = append(args, scopeID)
	}
	_, err = Store(s).exec(ctx, "DELETE FROM member_sessions WHERE id = "+s.ph(1)+clause, args...)
	return err
}

func (s SessionStore) DeleteByMemberID(ctx context.Context, memberID string) error {
	clause, scopeID, err := s.tenantClause(ctx, 2)
	if err != nil {
		return err
	}
	args := []any{memberID}
	if clause != "" {
		args = append(args, scopeID)
	}
	_, err = Store(s).exec(ctx, "DELETE FROM member_sessions WHERE member_id = "+s.ph(1)+clause, args...)
	return err
}

func (s SessionStore) ListByMemberID(ctx context.Context, memberID string) ([]members.Session, error) {
	clause, scopeID, err := s.tenantClause(ctx, 2)
	if err != nil {
		return nil, err
	}
	args := []any{memberID}
	if clause != "" {
		args = append(args, scopeID)
	}
	var result []members.Session
	err = Store(s).withinQuery(ctx, func(exec database.QueryExecutor) error {
		rows, err := exec.QueryContext(ctx, "SELECT id,member_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address FROM member_sessions WHERE member_id = "+s.ph(1)+clause+" ORDER BY last_seen_at DESC LIMIT 100", args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v members.Session
			var expires, created, lastSeen any
			if err := rows.Scan(&v.ID, &v.MemberID, &v.TokenHash, &expires, &created, &lastSeen, &v.UserAgent, &v.IPAddress); err != nil {
				return err
			}
			v.ExpiresAt = parseTime(expires)
			v.CreatedAt = parseTime(created)
			v.LastSeenAt = parseTime(lastSeen)
			result = append(result, v)
		}
		return rows.Err()
	})
	return result, err
}

func (s SessionStore) Touch(ctx context.Context, hash string, t time.Time) error {
	var touchVal any = t
	if s.Dialect == database.SQLite {
		touchVal = t.Format(time.RFC3339Nano)
	}
	clause, scopeID, err := s.tenantClause(ctx, 3)
	if err != nil {
		return err
	}
	args := []any{touchVal, hash}
	if clause != "" {
		args = append(args, scopeID)
	}
	_, err = Store(s).exec(ctx, "UPDATE member_sessions SET last_seen_at = "+s.ph(1)+" WHERE token_hash = "+s.ph(2)+clause, args...)
	return err
}

type AuditStore struct {
	DB                 *sql.DB
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s AuditStore) ListByTargetID(ctx context.Context, targetID string, limit int) ([]members.AuditEvent, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	ph := "?"
	if s.Dialect == database.Postgres {
		ph = "$1"
	}
	args := []any{targetID}
	query := "SELECT id, action, target_id, COALESCE(request_id, '') FROM audit_events WHERE target_id = " + ph
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return nil, err
		}
		if s.Dialect != database.Postgres {
			query += " AND tenant_id = ?"
		} else {
			query += " AND tenant_id = $2"
		}
		args = append(args, scope.TenantID)
	}
	query += " ORDER BY id DESC LIMIT " + fmt.Sprint(limit)
	var result []members.AuditEvent
	store := Store{DB: s.DB, Dialect: s.Dialect, RequireTenantScope: s.RequireTenantScope}
	err := store.withinQuery(ctx, func(exec database.QueryExecutor) error {
		rows, err := exec.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var ev members.AuditEvent
			if err := rows.Scan(&ev.ID, &ev.Action, &ev.TargetID, &ev.RequestID); err != nil {
				return err
			}
			result = append(result, ev)
		}
		return rows.Err()
	})
	return result, err
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
