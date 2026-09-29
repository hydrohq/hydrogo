package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type TokenLifetimeStore struct {
	DB      *sql.DB
	Dialect database.Dialect
}

func (s TokenLifetimeStore) Read(ctx context.Context) (auth.TokenLifetimes, bool, error) {
	if s.DB == nil {
		return auth.TokenLifetimes{}, false, errors.New("token lifetime database is required")
	}
	var minutes, hours int
	err := s.DB.QueryRowContext(ctx, `SELECT password_reset_minutes,invitation_hours FROM auth_token_lifetimes WHERE singleton_id=1`).Scan(&minutes, &hours)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.TokenLifetimes{}, false, nil
	}
	if err != nil {
		return auth.TokenLifetimes{}, false, err
	}
	value := auth.TokenLifetimes{PasswordReset: time.Duration(minutes) * time.Minute, Invitation: time.Duration(hours) * time.Hour}
	return value, true, value.Validate()
}

func (s TokenLifetimeStore) Write(ctx context.Context, value auth.TokenLifetimes) error {
	if s.DB == nil {
		return errors.New("token lifetime database is required")
	}
	if err := value.Validate(); err != nil {
		return err
	}
	minutes, hours := int(value.PasswordReset/time.Minute), int(value.Invitation/time.Hour)
	updated := time.Now().UTC()
	var stamp any = updated
	if s.Dialect == database.SQLite {
		stamp = updated.Format(time.RFC3339Nano)
	}
	query := `INSERT INTO auth_token_lifetimes(singleton_id,password_reset_minutes,invitation_hours,updated_at) VALUES(1,?,?,?) ON CONFLICT(singleton_id) DO UPDATE SET password_reset_minutes=excluded.password_reset_minutes,invitation_hours=excluded.invitation_hours,updated_at=excluded.updated_at`
	if s.Dialect == database.Postgres {
		query = `INSERT INTO auth_token_lifetimes(singleton_id,password_reset_minutes,invitation_hours,updated_at) VALUES(1,$1,$2,$3) ON CONFLICT(singleton_id) DO UPDATE SET password_reset_minutes=EXCLUDED.password_reset_minutes,invitation_hours=EXCLUDED.invitation_hours,updated_at=EXCLUDED.updated_at`
	}
	if s.Dialect == database.MySQL {
		query = `INSERT INTO auth_token_lifetimes(singleton_id,password_reset_minutes,invitation_hours,updated_at) VALUES(1,?,?,?) ON DUPLICATE KEY UPDATE password_reset_minutes=VALUES(password_reset_minutes),invitation_hours=VALUES(invitation_hours),updated_at=VALUES(updated_at)`
	}
	return database.NewTenantTransactor(s.DB, s.Dialect, false).Within(ctx, func(tx database.Executor) error {
		_, err := tx.ExecContext(ctx, query, minutes, hours, stamp)
		return err
	})
}
