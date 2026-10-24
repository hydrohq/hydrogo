// Package sqlstore provides a durable job lease store for SQLite and PostgreSQL.
package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
)

type Store struct {
	Dialect    database.Dialect
	MaxEntries int
	Tx         database.Transactor
}

// ListDeadLetters returns a bounded operator view of jobs that require review.
func (s Store) ListDeadLetters(ctx context.Context, db database.Executor, limit int) ([]jobs.Job, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 1
	}
	dead := "1"
	if s.Dialect == database.Postgres {
		dead = "TRUE"
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}
	rows, err := queryer.QueryContext(ctx, `SELECT id, type, payload, available_at, created_at, attempts, dead_lettered, COALESCE(last_error, '') FROM jobs WHERE dead_lettered = `+dead+` ORDER BY created_at, id LIMIT `+s.p(1), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]jobs.Job, 0, limit)
	for rows.Next() {
		var job jobs.Job
		var available, created any
		if err := rows.Scan(&job.ID, &job.Type, &job.Payload, &available, &created, &job.Attempts, &job.DeadLettered, &job.LastError); err != nil {
			return nil, err
		}
		job.AvailableAt, err = parseTime(available)
		if err != nil {
			return nil, err
		}
		job.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		result = append(result, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// JobSummary is the admin's JSON view of a job: jobs.Job has no JSON tags
// (nothing outside this package serialized it before), and it carries no
// lease state, so this type both tags every field and adds the lease.
type JobSummary struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Payload      string     `json:"payload"`
	AvailableAt  time.Time  `json:"available_at"`
	CreatedAt    time.Time  `json:"created_at"`
	Attempts     int        `json:"attempts"`
	DeadLettered bool       `json:"dead_lettered"`
	LastError    string     `json:"last_error,omitempty"`
	LeaseOwner   string     `json:"lease_owner,omitempty"`
	LeaseUntil   *time.Time `json:"lease_until,omitempty"`
}

// JobSummaryOf tags a bare jobs.Job (for example from ListDeadLetters) for
// the admin JSON API.
func JobSummaryOf(job jobs.Job) JobSummary {
	return JobSummary{
		ID:           job.ID,
		Type:         job.Type,
		Payload:      job.Payload,
		AvailableAt:  job.AvailableAt,
		CreatedAt:    job.CreatedAt,
		Attempts:     job.Attempts,
		DeadLettered: job.DeadLettered,
		LastError:    job.LastError,
	}
}

// List returns a bounded operator view of claimable and leased jobs (never
// dead-lettered ones; see ListDeadLetters for those).
func (s Store) List(ctx context.Context, db database.Executor, limit int) ([]JobSummary, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 1
	}
	dead := "0"
	if s.Dialect == database.Postgres {
		dead = "FALSE"
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("database executor does not support row queries")
	}
	rows, err := queryer.QueryContext(ctx, `SELECT id, type, payload, available_at, created_at, attempts, dead_lettered, COALESCE(last_error, ''), COALESCE(lease_owner, ''), lease_until FROM jobs WHERE dead_lettered = `+dead+` ORDER BY available_at, id LIMIT `+s.p(1), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]JobSummary, 0, limit)
	for rows.Next() {
		var summary JobSummary
		var available, created, leaseUntil any
		if err := rows.Scan(&summary.ID, &summary.Type, &summary.Payload, &available, &created, &summary.Attempts, &summary.DeadLettered, &summary.LastError, &summary.LeaseOwner, &leaseUntil); err != nil {
			return nil, err
		}
		summary.AvailableAt, err = parseTime(available)
		if err != nil {
			return nil, err
		}
		summary.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		if leaseUntil != nil {
			parsed, err := parseTime(leaseUntil)
			if err != nil {
				return nil, err
			}
			summary.LeaseUntil = &parsed
		}
		result = append(result, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteDeadLetter removes one explicitly selected dead-letter job.
func (s Store) DeleteDeadLetter(ctx context.Context, db database.Executor, id string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	dead := "1"
	if s.Dialect == database.Postgres {
		dead = "TRUE"
	}
	result, err := db.ExecContext(ctx, "DELETE FROM jobs WHERE id = "+s.p(1)+" AND dead_lettered = "+dead, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return jobs.ErrLeaseInvalid
	}
	return nil
}

// RequeueDeadLetter makes one explicitly selected dead-letter job claimable.
// The attempt count and last error remain available for operator audit.
func (s Store) RequeueDeadLetter(ctx context.Context, db database.Executor, id string, availableAt time.Time) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	dead := "1"
	if s.Dialect == database.Postgres {
		dead = "TRUE"
	}
	result, err := db.ExecContext(ctx, "UPDATE jobs SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, available_at = "+s.p(1)+", dead_lettered = "+s.boolLiteral(false)+" WHERE id = "+s.p(2)+" AND dead_lettered = "+dead, s.dbTime(availableAt.UTC()), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return jobs.ErrLeaseInvalid
	}
	return nil
}

// PurgeDeadLetters removes at most limit old dead-letter rows. It never
// touches leased or claimable jobs, and is intentionally bounded per call.
func (s Store) PurgeDeadLetters(ctx context.Context, db database.Executor, before time.Time, limit int) (int, error) {
	if err := database.RequireExecutor(db); err != nil {
		return 0, err
	}
	if before.IsZero() {
		return 0, errors.New("dead-letter purge cutoff is required")
	}
	if limit < 1 {
		return 0, nil
	}
	if limit > 10000 {
		limit = 10000
	}
	dead := "1"
	if s.Dialect == database.Postgres {
		dead = "TRUE"
	}
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return 0, errors.New("database executor does not support row queries")
	}
	rows, err := queryer.QueryContext(ctx, "SELECT id FROM jobs WHERE dead_lettered = "+dead+" AND created_at < "+s.p(1)+" ORDER BY created_at, id LIMIT "+s.p(2), s.dbTime(before.UTC()), limit)
	if err != nil {
		return 0, err
	}
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	removed := 0
	for _, id := range ids {
		result, err := db.ExecContext(ctx, "DELETE FROM jobs WHERE id = "+s.p(1)+" AND dead_lettered = "+dead, id)
		if err != nil {
			return removed, err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return removed, err
		}
		removed += int(count)
	}
	return removed, nil
}

func (s Store) Enqueue(ctx context.Context, db database.Executor, job jobs.Job) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if rawDB, ok := db.(*sql.DB); ok && s.MaxEntries > 0 {
		return database.NewTransactor(rawDB).Within(ctx, func(tx database.Executor) error {
			return s.enqueue(ctx, tx, job)
		})
	}
	return s.enqueue(ctx, db, job)
}

func (s Store) enqueue(ctx context.Context, db database.Executor, job jobs.Job) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	if strings.TrimSpace(job.ID) == "" || strings.TrimSpace(job.Type) == "" {
		return jobs.ErrJobConflict
	}
	if s.MaxEntries > 0 {
		// The singleton row is updated inside the enqueue transaction. PostgreSQL
		// serializes producers on the row lock; SQLite serializes the write.
		if _, err := db.ExecContext(ctx, "UPDATE job_queue_capacity_lock SET id = id WHERE id = "+s.p(1), 1); err != nil {
			return err
		}
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE dead_lettered = "+s.boolLiteral(false)).Scan(&count); err != nil {
			return err
		}
		if count >= s.MaxEntries {
			var existingType, existingPayload string
			err := db.QueryRowContext(ctx, "SELECT type, payload FROM jobs WHERE id = "+s.p(1), job.ID).Scan(&existingType, &existingPayload)
			if errors.Is(err, sql.ErrNoRows) {
				return jobs.ErrQueueFull
			}
			if err != nil {
				return err
			}
			if existingType != job.Type || existingPayload != job.Payload {
				return jobs.ErrJobConflict
			}
			return nil
		}
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.AvailableAt.IsZero() {
		job.AvailableAt = job.CreatedAt
	}
	_, err := db.ExecContext(ctx, `INSERT INTO jobs (id, type, payload, available_at, created_at, attempts) VALUES (`+s.p(1)+`, `+s.p(2)+`, `+s.p(3)+`, `+s.p(4)+`, `+s.p(5)+`, 0)`, job.ID, job.Type, job.Payload, s.dbTime(job.AvailableAt), s.dbTime(job.CreatedAt))
	if err != nil {
		if database.IsUniqueViolation(err) {
			var existingType, existingPayload string
			if lookupErr := db.QueryRowContext(ctx, "SELECT type, payload FROM jobs WHERE id = "+s.p(1), job.ID).Scan(&existingType, &existingPayload); lookupErr == nil && existingType == job.Type && existingPayload == job.Payload {
				return nil
			}
			return jobs.ErrJobConflict
		}
	}
	return err
}

func (s Store) Claim(ctx context.Context, db database.Executor, worker string, now time.Time, duration time.Duration) (jobs.Job, string, bool, error) {
	if err := database.RequireExecutor(db); err != nil {
		return jobs.Job{}, "", false, err
	}
	if strings.TrimSpace(worker) == "" || duration <= 0 {
		return jobs.Job{}, "", false, nil
	}
	var job jobs.Job
	var available, created any
	var attempts int
	deadLettered := "0"
	if s.Dialect == database.Postgres {
		deadLettered = "FALSE"
	}
	// restore_controls stores its flags as INTEGER in both dialects. Keep the
	// quarantine predicate numeric instead of using PostgreSQL's BOOLEAN literal.
	err := db.QueryRowContext(ctx, `SELECT id, type, payload, available_at, created_at, attempts, dead_lettered, COALESCE(last_error, '') FROM jobs WHERE available_at <= `+s.p(1)+` AND (lease_until IS NULL OR lease_until <= `+s.p(2)+`) AND dead_lettered = `+deadLettered+` AND NOT EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND egress_quarantined = 1) ORDER BY available_at, id LIMIT 1`, s.dbTime(now), s.dbTime(now)).Scan(&job.ID, &job.Type, &job.Payload, &available, &created, &attempts, &job.DeadLettered, &job.LastError)
	if errors.Is(err, sql.ErrNoRows) {
		return jobs.Job{}, "", false, nil
	}
	if err != nil {
		return jobs.Job{}, "", false, err
	}
	job.AvailableAt, err = parseTime(available)
	if err != nil {
		return jobs.Job{}, "", false, err
	}
	job.CreatedAt, err = parseTime(created)
	if err != nil {
		return jobs.Job{}, "", false, err
	}
	job.Attempts = attempts + 1
	token := newToken(worker, job.ID, now)
	result, err := db.ExecContext(ctx, `UPDATE jobs SET lease_owner = `+s.p(1)+`, lease_token = `+s.p(2)+`, lease_until = `+s.p(3)+`, attempts = attempts + 1 WHERE id = `+s.p(4)+` AND (lease_until IS NULL OR lease_until <= `+s.p(5)+`) AND dead_lettered = `+deadLettered+` AND NOT EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND egress_quarantined = 1)`, worker, token, s.dbTime(now.Add(duration)), job.ID, s.dbTime(now))
	if err != nil {
		return jobs.Job{}, "", false, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return jobs.Job{}, "", false, err
	}
	if updated != 1 {
		return jobs.Job{}, "", false, nil
	}
	return job, token, true, nil
}

func (s Store) Ack(ctx context.Context, db database.Executor, id, worker, token string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, "DELETE FROM jobs WHERE id = "+s.p(1)+" AND lease_owner = "+s.p(2)+" AND lease_token = "+s.p(3), id, worker, token)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return jobs.ErrLeaseInvalid
	}
	return nil
}

func (s Store) Nack(ctx context.Context, db database.Executor, id, worker, token string, retryAt time.Time) error {
	return s.NackWithError(ctx, db, id, worker, token, retryAt, "")
}

func (s Store) NackWithError(ctx context.Context, db database.Executor, id, worker, token string, retryAt time.Time, reason string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, "UPDATE jobs SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, available_at = "+s.p(1)+", last_error = "+s.p(2)+" WHERE id = "+s.p(3)+" AND lease_owner = "+s.p(4)+" AND lease_token = "+s.p(5), s.dbTime(retryAt.UTC()), jobs.BoundError(reason), id, worker, token)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return jobs.ErrLeaseInvalid
	}
	return nil
}

func (s Store) DeadLetter(ctx context.Context, db database.Executor, id, worker, token, reason string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	dead := "1"
	if s.Dialect == database.Postgres {
		dead = "TRUE"
	}
	result, err := db.ExecContext(ctx, "UPDATE jobs SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, dead_lettered = "+dead+", last_error = "+s.p(1)+" WHERE id = "+s.p(2)+" AND lease_owner = "+s.p(3)+" AND lease_token = "+s.p(4), jobs.BoundError(reason), id, worker, token)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return jobs.ErrLeaseInvalid
	}
	return nil
}

func (s Store) p(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func (s Store) boolLiteral(value bool) string {
	if s.Dialect == database.Postgres {
		if value {
			return "TRUE"
		}
		return "FALSE"
	}
	if value {
		return "1"
	}
	return "0"
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func (s Store) dbTime(value time.Time) any {
	if s.Dialect == database.MySQL {
		return value.UTC()
	}
	return formatTime(value)
}

func parseTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v.UTC(), nil
	case string:
		return time.Parse(time.RFC3339Nano, v)
	case []byte:
		return time.Parse(time.RFC3339Nano, string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported stored time value type %T", value)
	}
}

func newToken(worker, id string, now time.Time) string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return worker + ":" + id + ":" + formatTime(now)
	}
	return worker + ":" + id + ":" + hex.EncodeToString(random)
}
