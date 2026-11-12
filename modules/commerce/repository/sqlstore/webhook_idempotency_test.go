package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockErrorExecutor struct {
	err error
}

func (m mockErrorExecutor) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, m.err
}

func (m mockErrorExecutor) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, m.err
}

func (m mockErrorExecutor) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

func TestRecordPaymentNotificationHandlesPostgresAndSQLiteUniqueViolation(t *testing.T) {
	ctx := context.Background()

	// 1. PostgreSQL error code 23505 via pgconn.PgError
	storePG := Store{Dialect: database.Postgres}
	pgExec := mockErrorExecutor{err: &pgconn.PgError{Code: "23505"}}
	isNew, err := storePG.RecordPaymentNotification(ctx, pgExec, "paytr", "dup-event-pg")
	if err != nil {
		t.Fatalf("expected nil error for pgx 23505 duplicate, got %v", err)
	}
	if isNew {
		t.Fatalf("expected isNew=false for pgx 23505 duplicate, got true")
	}

	// 2. SQLite UNIQUE constraint failed error
	storeSQLite := Store{Dialect: database.SQLite}
	sqliteExec := mockErrorExecutor{err: errors.New("constraint failed: UNIQUE constraint failed: commerce_payment_notifications.provider, commerce_payment_notifications.provider_event_id")}
	isNew, err = storeSQLite.RecordPaymentNotification(ctx, sqliteExec, "iyzico", "dup-event-sqlite")
	if err != nil {
		t.Fatalf("expected nil error for sqlite unique constraint duplicate, got %v", err)
	}
	if isNew {
		t.Fatalf("expected isNew=false for sqlite duplicate, got true")
	}

	// 3. Unrelated database error must still return error (not swallowed)
	unrelatedExec := mockErrorExecutor{err: errors.New("connection reset by peer")}
	_, err = storePG.RecordPaymentNotification(ctx, unrelatedExec, "paytr", "err-event")
	if err == nil {
		t.Fatalf("expected error for unrelated failure, got nil")
	}
}
