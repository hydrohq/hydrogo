package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Store struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s Store) p(n int) string {
	if s.Dialect == database.SQLite {
		return "?"
	}
	return "$" + strconv.Itoa(n)
}

func (s Store) tenantFilter(ctx context.Context, position int) (string, any, error) {
	if !s.RequireTenantScope {
		return " AND tenant_id = " + s.p(position), "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", nil, err
	}
	return " AND tenant_id = " + s.p(position), scope.TenantID, nil
}

func (s Store) UpsertSetting(ctx context.Context, tx database.Executor, value site.Setting) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := value.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if value.UpdatedAt.IsZero() {
		now = time.Now().UTC().Format(time.RFC3339Nano)
	}
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	query := `INSERT INTO site_settings (tenant_id,key,value,is_public,updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `) ON CONFLICT (tenant_id,key) DO UPDATE SET value=EXCLUDED.value,is_public=EXCLUDED.is_public,updated_at=EXCLUDED.updated_at`
	args := []any{scopeID, value.Key, value.Value, value.Public, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO site_settings (tenant_id,key,value,is_public,updated_at) VALUES (?,?,?,?,?) ON CONFLICT(tenant_id,key) DO UPDATE SET value=excluded.value,is_public=excluded.is_public,updated_at=excluded.updated_at`
		args[3] = boolInt(value.Public)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) ListPublicSettings(ctx context.Context, db database.Executor) ([]site.Setting, error) {
	filter, scopeID, err := s.tenantFilter(ctx, 1)
	if err != nil {
		return nil, err
	}
	query := `SELECT key,value,is_public,updated_at FROM site_settings WHERE is_public = TRUE` + filter + ` ORDER BY key LIMIT 1000`
	if s.Dialect == database.SQLite {
		query = `SELECT key,value,is_public,updated_at FROM site_settings WHERE is_public = 1` + filter + ` ORDER BY key LIMIT 1000`
	}
	args := []any{}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := queryContext(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.Setting, 0)
	for rows.Next() {
		var value site.Setting
		var public any
		var updated any
		if err := rows.Scan(&value.Key, &value.Value, &public, &updated); err != nil {
			return nil, err
		}
		value.Public = parseBool(public)
		value.UpdatedAt = parseTime(updated)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) ReplaceNavigation(ctx context.Context, tx database.Executor, locale string, items []site.NavigationItem) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return err
	}
	deleteQuery := `DELETE FROM navigation_items WHERE locale = $1` + filter
	if s.Dialect == database.SQLite {
		deleteQuery = `DELETE FROM navigation_items WHERE locale = ?` + filter
	}
	deleteArgs := []any{locale}
	if filter != "" {
		deleteArgs = append(deleteArgs, scopeID)
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return err
	}
	for _, item := range items {
		query := `INSERT INTO navigation_items (tenant_id,id,locale,label,href,position,is_visible,updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `)`
		args := []any{scopeID, item.ID, locale, item.Label, item.Href, item.Position, item.Visible, item.UpdatedAt.UTC().Format(time.RFC3339Nano)}
		if item.UpdatedAt.IsZero() {
			args[7] = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if s.Dialect == database.SQLite {
			query = `INSERT INTO navigation_items (tenant_id,id,locale,label,href,position,is_visible,updated_at) VALUES (?,?,?,?,?,?,?,?)`
			args[6] = boolInt(item.Visible)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ListNavigation(ctx context.Context, db database.Executor, locale string) ([]site.NavigationItem, error) {
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	query := `SELECT id,locale,label,href,position,is_visible,updated_at FROM navigation_items WHERE locale = $1 AND is_visible = TRUE` + filter + ` ORDER BY position,id LIMIT 500`
	if s.Dialect == database.SQLite {
		query = `SELECT id,locale,label,href,position,is_visible,updated_at FROM navigation_items WHERE locale = ? AND is_visible = 1` + filter + ` ORDER BY position,id LIMIT 500`
	}
	args := []any{locale}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := queryContext(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.NavigationItem, 0)
	for rows.Next() {
		var item site.NavigationItem
		var visible any
		var updated any
		if err := rows.Scan(&item.ID, &item.Locale, &item.Label, &item.Href, &item.Position, &visible, &updated); err != nil {
			return nil, err
		}
		item.Visible = parseBool(visible)
		item.UpdatedAt = parseTime(updated)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) PutRedirect(ctx context.Context, tx database.Executor, value site.Redirect) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	now := value.CreatedAt.UTC().Format(time.RFC3339Nano)
	if value.CreatedAt.IsZero() {
		now = time.Now().UTC().Format(time.RFC3339Nano)
	}
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	query := `INSERT INTO page_redirects (tenant_id,source_path,target_path,status,created_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `) ON CONFLICT (tenant_id,source_path) DO UPDATE SET target_path=EXCLUDED.target_path,status=EXCLUDED.status`
	args := []any{scopeID, value.SourcePath, value.TargetPath, value.Status, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO page_redirects (tenant_id,source_path,target_path,status,created_at) VALUES (?,?,?,?,?) ON CONFLICT(tenant_id,source_path) DO UPDATE SET target_path=excluded.target_path,status=excluded.status`
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) GetRedirect(ctx context.Context, db database.Executor, source string) (site.Redirect, error) {
	if err := database.RequireExecutor(db); err != nil {
		return site.Redirect{}, err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return site.Redirect{}, err
	}
	query := `SELECT source_path,target_path,status,created_at FROM page_redirects WHERE source_path = $1` + filter
	if s.Dialect == database.SQLite {
		query = `SELECT source_path,target_path,status,created_at FROM page_redirects WHERE source_path = ?` + filter
	}
	var value site.Redirect
	var created any
	args := []any{source}
	if filter != "" {
		args = append(args, scopeID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&value.SourcePath, &value.TargetPath, &value.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return site.Redirect{}, sql.ErrNoRows
	}
	value.CreatedAt = parseTime(created)
	return value, err
}

func (s Store) CreateSubmission(ctx context.Context, tx database.Executor, value site.Submission) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	query := `INSERT INTO form_inbox (tenant_id,id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `)`
	created := value.CreatedAt.UTC().Format(time.RFC3339Nano)
	if value.CreatedAt.IsZero() {
		created = time.Now().UTC().Format(time.RFC3339Nano)
	}
	status := value.Status
	if status == "" {
		status = "new"
	}
	leadStage := value.LeadStage
	if leadStage == "" {
		leadStage = "new"
	}
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	args := []any{scopeID, value.ID, value.FormKey, value.Email, value.Payload, value.SourceIPHash, status, created, value.IdempotencyKey, value.RequestHash, leadStage, value.AssignedTo}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) FindSubmissionByIdempotency(ctx context.Context, db database.Executor, formKey, key string) (site.Submission, bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return site.Submission{}, false, err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return site.Submission{}, false, err
	}
	query := `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to FROM form_inbox WHERE form_key = $1 AND idempotency_key = $2` + filter + ` LIMIT 1`
	if s.Dialect == database.SQLite {
		query = `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to FROM form_inbox WHERE form_key = ? AND idempotency_key = ?` + filter + ` LIMIT 1`
	}
	var value site.Submission
	var created any
	args := []any{formKey, key}
	if filter != "" {
		args = append(args, scopeID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&value.ID, &value.FormKey, &value.Email, &value.Payload, &value.SourceIPHash, &value.Status, &created, &value.IdempotencyKey, &value.RequestHash, &value.LeadStage, &value.AssignedTo)
	if errors.Is(err, sql.ErrNoRows) {
		return site.Submission{}, false, nil
	}
	value.CreatedAt = parseTime(created)
	return value, err == nil, err
}

func (s Store) ListSubmissions(ctx context.Context, db database.Executor, formKey string, limit int) ([]site.Submission, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return nil, err
	}
	query := `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to FROM form_inbox WHERE form_key = $1` + filter + ` ORDER BY created_at DESC LIMIT $2`
	if s.Dialect == database.SQLite {
		query = `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to FROM form_inbox WHERE form_key = ?` + filter + ` ORDER BY created_at DESC LIMIT ?`
	}
	args := []any{formKey, limit}
	if filter != "" {
		if s.Dialect == database.Postgres {
			args = append(args, scopeID)
		} else {
			args = []any{formKey, scopeID, limit}
		}
	}
	rows, err := queryContext(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.Submission, 0)
	for rows.Next() {
		var value site.Submission
		var created any
		if err := rows.Scan(&value.ID, &value.FormKey, &value.Email, &value.Payload, &value.SourceIPHash, &value.Status, &created, &value.IdempotencyKey, &value.RequestHash, &value.LeadStage, &value.AssignedTo); err != nil {
			return nil, err
		}
		value.CreatedAt = parseTime(created)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) UpdateLead(ctx context.Context, tx database.Executor, id, stage, assignedTo string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return err
	}
	query := `UPDATE form_inbox SET lead_status = $1, assigned_to = $2 WHERE id = $3` + filter
	if s.Dialect == database.SQLite {
		query = `UPDATE form_inbox SET lead_status = ?, assigned_to = ? WHERE id = ?` + filter
	}
	args := []any{stage, assignedTo, id}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Store) CreateLeadNote(ctx context.Context, tx database.Executor, note site.LeadNote) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	tenantID := ""
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return err
		}
		tenantID = scope.TenantID
	}
	if note.CreatedAt.IsZero() {
		note.CreatedAt = time.Now().UTC()
	}
	query := `INSERT INTO form_lead_notes (id,tenant_id,submission_id,author_id,note,created_at) SELECT $1,$2,$3,$4,$5,$6 WHERE EXISTS (SELECT 1 FROM form_inbox WHERE id = $3 AND tenant_id = $2)`
	args := []any{note.ID, tenantID, note.SubmissionID, note.AuthorID, strings.TrimSpace(note.Note), note.CreatedAt.UTC()}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO form_lead_notes (id,tenant_id,submission_id,author_id,note,created_at) SELECT ?,?,?,?,?,? WHERE EXISTS (SELECT 1 FROM form_inbox WHERE id = ? AND tenant_id = ?)`
		args = []any{note.ID, tenantID, note.SubmissionID, note.AuthorID, strings.TrimSpace(note.Note), note.CreatedAt.UTC().Format(time.RFC3339Nano), note.SubmissionID, tenantID}
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Store) ListLeadNotes(ctx context.Context, db database.QueryExecutor, submissionID string, limit int) ([]site.LeadNote, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return nil, err
	}
	query := `SELECT id,tenant_id,submission_id,author_id,note,created_at FROM form_lead_notes WHERE submission_id = $1` + filter + ` ORDER BY created_at,id LIMIT $2`
	args := []any{submissionID, limit}
	if s.Dialect == database.SQLite {
		query = `SELECT id,tenant_id,submission_id,author_id,note,created_at FROM form_lead_notes WHERE submission_id = ?` + filter + ` ORDER BY created_at,id LIMIT ?`
	}
	if filter != "" {
		if s.Dialect == database.Postgres {
			args = []any{submissionID, scopeID, limit}
		} else {
			args = []any{submissionID, scopeID, limit}
		}
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.LeadNote, 0, limit)
	for rows.Next() {
		var item site.LeadNote
		var created any
		if err := rows.Scan(&item.ID, &item.TenantID, &item.SubmissionID, &item.AuthorID, &item.Note, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) UpdateSubmissionStatus(ctx context.Context, tx database.Executor, id, status string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 3)
	if err != nil {
		return err
	}
	query := `UPDATE form_inbox SET status = $1 WHERE id = $2` + filter
	if s.Dialect == database.SQLite {
		query = `UPDATE form_inbox SET status = ? WHERE id = ?` + filter
	}
	args := []any{status, id}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Store) PurgeSubmissionsBefore(ctx context.Context, tx database.Executor, before time.Time) (int64, error) {
	if err := database.RequireExecutor(tx); err != nil {
		return 0, err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return 0, err
	}
	query := `DELETE FROM form_inbox WHERE created_at < $1` + filter
	arg := any(before)
	if s.Dialect == database.SQLite {
		query = `DELETE FROM form_inbox WHERE created_at < ?` + filter
		arg = before.Format(time.RFC3339Nano)
	}
	args := []any{arg}
	if filter != "" {
		args = append(args, scopeID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func parseBool(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case int64:
		return value != 0
	case int:
		return value != 0
	case []byte:
		return string(value) == "1" || strings.EqualFold(string(value), "true")
	case string:
		return value == "1" || strings.EqualFold(value, "true")
	default:
		return false
	}
}

type queryExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryContext(ctx context.Context, db database.Executor, query string, args ...any) (*sql.Rows, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	queryer, ok := db.(queryExecutor)
	if !ok {
		return nil, fmt.Errorf("database executor does not support row queries")
	}
	return queryer.QueryContext(ctx, query, args...)
}

func parseTime(value any) time.Time {
	switch value := value.(type) {
	case time.Time:
		return value.UTC()
	case string:
		parsed, _ := time.Parse(time.RFC3339Nano, value)
		return parsed.UTC()
	case []byte:
		parsed, _ := time.Parse(time.RFC3339Nano, string(value))
		return parsed.UTC()
	default:
		return time.Time{}
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
