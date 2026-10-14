package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (s Store) CreateSubmission(ctx context.Context, tx database.Executor, value site.Submission) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	query := `INSERT INTO form_inbox (tenant_id,id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to,location_country_code,location_city,location_local) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,` + s.p(7) + `,` + s.p(8) + `,` + s.p(9) + `,` + s.p(10) + `,` + s.p(11) + `,` + s.p(12) + `,` + s.p(13) + `,` + s.p(14) + `,` + s.p(15) + `)`
	createdAt := value.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	var created any = createdAt
	if s.Dialect == database.SQLite {
		created = createdAt.Format(time.RFC3339Nano)
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
	args := []any{scopeID, value.ID, value.FormKey, value.Email, value.Payload, value.SourceIPHash, status, created, value.IdempotencyKey, value.RequestHash, leadStage, value.AssignedTo, value.LocationCountryCode, value.LocationCity, value.LocationLocal}
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
	query := `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to,location_country_code,location_city,location_local FROM form_inbox WHERE form_key = $1 AND idempotency_key = $2` + filter + ` LIMIT 1`
	if s.Dialect != database.Postgres {
		query = `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to,location_country_code,location_city,location_local FROM form_inbox WHERE form_key = ? AND idempotency_key = ?` + filter + ` LIMIT 1`
	}
	var value site.Submission
	var created any
	args := []any{formKey, key}
	if filter != "" {
		args = append(args, scopeID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&value.ID, &value.FormKey, &value.Email, &value.Payload, &value.SourceIPHash, &value.Status, &created, &value.IdempotencyKey, &value.RequestHash, &value.LeadStage, &value.AssignedTo, &value.LocationCountryCode, &value.LocationCity, &value.LocationLocal)
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
	query := `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to,location_country_code,location_city,location_local FROM form_inbox WHERE form_key = $1 AND deleted_at IS NULL` + filter + ` ORDER BY created_at DESC LIMIT $2`
	if s.Dialect != database.Postgres {
		query = `SELECT id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash,lead_status,assigned_to,location_country_code,location_city,location_local FROM form_inbox WHERE form_key = ? AND deleted_at IS NULL` + filter + ` ORDER BY created_at DESC LIMIT ?`
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
		if err := rows.Scan(&value.ID, &value.FormKey, &value.Email, &value.Payload, &value.SourceIPHash, &value.Status, &created, &value.IdempotencyKey, &value.RequestHash, &value.LeadStage, &value.AssignedTo, &value.LocationCountryCode, &value.LocationCity, &value.LocationLocal); err != nil {
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
	query := `UPDATE form_inbox SET lead_status = $1, assigned_to = $2 WHERE id = $3 AND deleted_at IS NULL` + filter
	if s.Dialect != database.Postgres {
		query = `UPDATE form_inbox SET lead_status = ?, assigned_to = ? WHERE id = ? AND deleted_at IS NULL` + filter
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
	query := `INSERT INTO form_lead_notes (id,tenant_id,submission_id,author_id,note,created_at) SELECT $1,$2,$3,$4,$5,$6 WHERE EXISTS (SELECT 1 FROM form_inbox WHERE id = $3 AND tenant_id = $2 AND deleted_at IS NULL)`
	createdAt := note.CreatedAt.UTC()
	var createdArg any = createdAt
	if s.Dialect == database.SQLite {
		createdArg = createdAt.Format(time.RFC3339Nano)
	}
	args := []any{note.ID, tenantID, note.SubmissionID, note.AuthorID, strings.TrimSpace(note.Note), createdArg}
	if s.Dialect != database.Postgres {
		query = `INSERT INTO form_lead_notes (id,tenant_id,submission_id,author_id,note,created_at) SELECT ?,?,?,?,?,? WHERE EXISTS (SELECT 1 FROM form_inbox WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL)`
		args = []any{note.ID, tenantID, note.SubmissionID, note.AuthorID, strings.TrimSpace(note.Note), createdArg, note.SubmissionID, tenantID}
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
	query := `SELECT id,tenant_id,submission_id,author_id,note,created_at FROM form_lead_notes WHERE submission_id = $1` + filter + ` ORDER BY created_at,id LIMIT ` + s.p(3)
	args := []any{submissionID, limit}
	if s.Dialect != database.Postgres {
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
	query := `UPDATE form_inbox SET status = $1 WHERE id = $2 AND deleted_at IS NULL` + filter
	if s.Dialect != database.Postgres {
		query = `UPDATE form_inbox SET status = ? WHERE id = ? AND deleted_at IS NULL` + filter
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
	if s.Dialect != database.Postgres {
		query = `DELETE FROM form_inbox WHERE created_at < ?` + filter
		if s.Dialect == database.SQLite {
			arg = before.Format(time.RFC3339Nano)
		}
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
