package notify

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type NotificationRecord struct {
	ID            string     `json:"id"`
	Channel       string     `json:"channel"`
	Payload       string     `json:"payload"`
	Attempts      int        `json:"attempts"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	LastError     string     `json:"last_error"`
	TenantID      string     `json:"tenant_id"`
}

type OutboxStore struct {
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s OutboxStore) p(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s OutboxStore) dbTime(t time.Time) any {
	if s.Dialect == database.MySQL {
		return t.UTC()
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (s OutboxStore) parseTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v.UTC(), nil
	case string:
		return time.Parse(time.RFC3339Nano, v)
	case []byte:
		return time.Parse(time.RFC3339Nano, string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported stored time type %T", value)
	}
}

func (s OutboxStore) tenantID(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func generateOutboxID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "nob_" + hex.EncodeToString(b), nil
}

func (s OutboxStore) Enqueue(ctx context.Context, db database.Executor, channel string, msg Message) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	tid, err := s.tenantID(ctx)
	if err != nil {
		return err
	}
	id, err := generateOutboxID()
	if err != nil {
		return fmt.Errorf("generate notification id: %w", err)
	}
	payloadBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	now := time.Now().UTC()
	query := `INSERT INTO notification_outbox (id, channel, payload, attempts, next_attempt_at, last_error, tenant_id) VALUES (` +
		s.p(1) + `, ` + s.p(2) + `, ` + s.p(3) + `, 0, ` + s.p(4) + `, '', ` + s.p(5) + `)`
	_, err = db.ExecContext(ctx, query, id, channel, string(payloadBytes), s.dbTime(now), tid)
	return err
}

func (s OutboxStore) ClaimDue(ctx context.Context, db database.QueryExecutor, now time.Time, maxAttempts, limit int) ([]NotificationRecord, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	var (
		rows *sql.Rows
		err  error
	)
	if s.RequireTenantScope {
		tid, err := s.tenantID(ctx)
		if err != nil {
			return nil, err
		}
		query := `SELECT id, channel, payload, attempts, next_attempt_at, sent_at, last_error, tenant_id
			FROM notification_outbox
			WHERE sent_at IS NULL AND next_attempt_at <= ` + s.p(1) + ` AND attempts < ` + s.p(2) + ` AND tenant_id = ` + s.p(3) + `
			ORDER BY next_attempt_at ASC LIMIT ` + s.p(4)
		rows, err = db.QueryContext(ctx, query, s.dbTime(now), maxAttempts, tid, limit)
		if err != nil {
			return nil, err
		}
	} else {
		query := `SELECT id, channel, payload, attempts, next_attempt_at, sent_at, last_error, tenant_id
			FROM notification_outbox
			WHERE sent_at IS NULL AND next_attempt_at <= ` + s.p(1) + ` AND attempts < ` + s.p(2) + `
			ORDER BY next_attempt_at ASC LIMIT ` + s.p(3)
		rows, err = db.QueryContext(ctx, query, s.dbTime(now), maxAttempts, limit)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	var records []NotificationRecord
	for rows.Next() {
		var (
			rec      NotificationRecord
			nextRaw  any
			sentRaw  any
			lastErr  sql.NullString
			tenantID sql.NullString
		)
		if err := rows.Scan(&rec.ID, &rec.Channel, &rec.Payload, &rec.Attempts, &nextRaw, &sentRaw, &lastErr, &tenantID); err != nil {
			return nil, err
		}
		if nextRaw != nil {
			t, err := s.parseTime(nextRaw)
			if err != nil {
				return nil, err
			}
			rec.NextAttemptAt = t
		}
		if sentRaw != nil {
			t, err := s.parseTime(sentRaw)
			if err != nil {
				return nil, err
			}
			rec.SentAt = &t
		}
		rec.LastError = lastErr.String
		rec.TenantID = tenantID.String
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (s OutboxStore) MarkSent(ctx context.Context, db database.Executor, id string, sentAt time.Time) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	query := `UPDATE notification_outbox SET sent_at = ` + s.p(1) + ` WHERE id = ` + s.p(2)
	_, err := db.ExecContext(ctx, query, s.dbTime(sentAt), id)
	return err
}

func (s OutboxStore) RecordRetry(ctx context.Context, db database.Executor, id string, attempts int, nextAttempt time.Time, lastErr string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	query := `UPDATE notification_outbox SET attempts = ` + s.p(1) + `, next_attempt_at = ` + s.p(2) + `, last_error = ` + s.p(3) + ` WHERE id = ` + s.p(4)
	_, err := db.ExecContext(ctx, query, attempts, s.dbTime(nextAttempt), lastErr, id)
	return err
}

func (s OutboxStore) MarkDeadLetter(ctx context.Context, db database.Executor, id string, attempts int, lastErr string) error {
	if err := database.RequireExecutor(db); err != nil {
		return err
	}
	deadErr := "dead_letter: " + lastErr
	farFuture := time.Now().UTC().Add(365 * 24 * time.Hour)
	query := `UPDATE notification_outbox SET attempts = ` + s.p(1) + `, next_attempt_at = ` + s.p(2) + `, last_error = ` + s.p(3) + ` WHERE id = ` + s.p(4)
	_, err := db.ExecContext(ctx, query, attempts, s.dbTime(farFuture), deadErr, id)
	return err
}

type OutboxSender struct {
	Store    OutboxStore
	DB       database.Executor
	Channel  string
	Fallback Sender
}

func (s OutboxSender) Send(ctx context.Context, msg Message) error {
	if s.DB != nil {
		channel := s.Channel
		if channel == "" {
			channel = "telegram"
		}
		return s.Store.Enqueue(ctx, s.DB, channel, msg)
	}
	if s.Fallback != nil {
		return s.Fallback.Send(ctx, msg)
	}
	return errors.New("outbox sender has no db executor or fallback")
}

type Worker struct {
	DB                  database.QueryExecutor
	Store               OutboxStore
	Sender              Sender
	MaxDeliveryAttempts int
	InitialBackoff      time.Duration
	MaxBackoff          time.Duration
	Logger              *slog.Logger
	TenantScopes        []tenant.Scope
}

func (w Worker) maxAttempts() int {
	if w.MaxDeliveryAttempts <= 0 {
		return 5
	}
	return w.MaxDeliveryAttempts
}

func (w Worker) initialBackoff() time.Duration {
	if w.InitialBackoff <= 0 {
		return 100 * time.Millisecond
	}
	return w.InitialBackoff
}

func (w Worker) maxBackoff() time.Duration {
	if w.MaxBackoff <= 0 {
		return 1 * time.Hour
	}
	return w.MaxBackoff
}

func (w Worker) calculateBackoff(attempts int) time.Duration {
	backoff := w.initialBackoff()
	for i := 1; i < attempts; i++ {
		backoff *= 2
		if backoff >= w.maxBackoff() {
			return w.maxBackoff()
		}
	}
	return backoff
}

func (w Worker) RunOnce(ctx context.Context, now time.Time) (int, error) {
	if len(w.TenantScopes) > 0 {
		total := 0
		for _, scope := range w.TenantScopes {
			scopedCtx := tenant.WithScope(ctx, scope)
			n, err := w.runScopedOnce(scopedCtx, now)
			if err != nil {
				return total, err
			}
			total += n
		}
		return total, nil
	}
	return w.runScopedOnce(ctx, now)
}

func (w Worker) runScopedOnce(ctx context.Context, now time.Time) (int, error) {
	records, err := w.Store.ClaimDue(ctx, w.DB, now, w.maxAttempts(), 50)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, rec := range records {
		var msg Message
		if err := json.Unmarshal([]byte(rec.Payload), &msg); err != nil {
			if w.Logger != nil {
				w.Logger.ErrorContext(ctx, "failed to unmarshal notification payload", "id", rec.ID, "error", err)
			}
			_ = w.Store.MarkDeadLetter(ctx, w.DB, rec.ID, rec.Attempts+1, "unmarshal error: "+err.Error())
			continue
		}
		sendErr := w.Sender.Send(ctx, msg)
		if sendErr == nil {
			if err := w.Store.MarkSent(ctx, w.DB, rec.ID, now); err != nil && w.Logger != nil {
				w.Logger.WarnContext(ctx, "failed to mark notification sent", "id", rec.ID, "error", err)
			}
			processed++
			continue
		}

		attempts := rec.Attempts + 1
		if attempts >= w.maxAttempts() {
			_ = w.Store.MarkDeadLetter(ctx, w.DB, rec.ID, attempts, sendErr.Error())
			if w.Logger != nil {
				w.Logger.ErrorContext(ctx, "notification permanently failed, marked as dead-letter",
					"id", rec.ID, "channel", rec.Channel, "attempts", attempts, "error", sendErr)
			}
		} else {
			backoff := w.calculateBackoff(attempts)
			nextAttempt := now.Add(backoff)
			_ = w.Store.RecordRetry(ctx, w.DB, rec.ID, attempts, nextAttempt, sendErr.Error())
			if w.Logger != nil {
				w.Logger.WarnContext(ctx, "notification delivery failed, scheduled retry",
					"id", rec.ID, "attempt", attempts, "next_attempt_at", nextAttempt, "error", sendErr)
			}
		}
		processed++
	}
	return processed, nil
}

func (w Worker) Run(ctx context.Context, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, _ = w.RunOnce(ctx, time.Now().UTC())
		}
	}
}
