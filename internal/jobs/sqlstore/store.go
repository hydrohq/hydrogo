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

func (s Store) Enqueue(ctx context.Context, db database.Executor, job jobs.Job) error {
	if strings.TrimSpace(job.ID) == "" || strings.TrimSpace(job.Type) == "" {
		return jobs.ErrJobConflict
	}
	if s.MaxEntries > 0 {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs").Scan(&count); err != nil {
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
	_, err := db.ExecContext(ctx, `INSERT INTO jobs (id, type, payload, available_at, created_at, attempts) VALUES (`+s.p(1)+`, `+s.p(2)+`, `+s.p(3)+`, `+s.p(4)+`, `+s.p(5)+`, 0)`, job.ID, job.Type, job.Payload, formatTime(job.AvailableAt), formatTime(job.CreatedAt))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
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
	if strings.TrimSpace(worker) == "" || duration <= 0 {
		return jobs.Job{}, "", false, nil
	}
	var job jobs.Job
	var available, created string
	var attempts int
	err := db.QueryRowContext(ctx, `SELECT id, type, payload, available_at, created_at, attempts FROM jobs WHERE available_at <= `+s.p(1)+` AND (lease_until IS NULL OR lease_until <= `+s.p(2)+`) AND NOT EXISTS (SELECT 1 FROM restore_controls WHERE id = 1 AND egress_quarantined = 1) ORDER BY available_at, id LIMIT 1`, formatTime(now), formatTime(now)).Scan(&job.ID, &job.Type, &job.Payload, &available, &created, &attempts)
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
	result, err := db.ExecContext(ctx, `UPDATE jobs SET lease_owner = `+s.p(1)+`, lease_token = `+s.p(2)+`, lease_until = `+s.p(3)+`, attempts = attempts + 1 WHERE id = `+s.p(4)+` AND (lease_until IS NULL OR lease_until <= `+s.p(5)+`)`, worker, token, formatTime(now.Add(duration)), job.ID, formatTime(now))
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
	result, err := db.ExecContext(ctx, "UPDATE jobs SET lease_owner = NULL, lease_token = NULL, lease_until = NULL, available_at = "+s.p(1)+" WHERE id = "+s.p(2)+" AND lease_owner = "+s.p(3)+" AND lease_token = "+s.p(4), formatTime(retryAt.UTC()), id, worker, token)
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

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }

func newToken(worker, id string, now time.Time) string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return worker + ":" + id + ":" + formatTime(now)
	}
	return worker + ":" + id + ":" + hex.EncodeToString(random)
}
