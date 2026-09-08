package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// ApplyRestoreControls revokes restored sessions and persists the quarantine
// state before any worker or public traffic is enabled.
func ApplyRestoreControls(ctx context.Context, db *sql.DB, dialect database.Dialect, controls RestoreControls) error {
	if db == nil {
		return errors.New("restore database is required")
	}
	if !controls.SessionsRevoked || !controls.EgressQuarantined || !controls.OldWritesBlocked {
		return errors.New("restore controls are incomplete")
	}
	return database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM sessions"); err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		query := "INSERT INTO restore_controls (id, sessions_revoked, egress_quarantined, old_writes_blocked, activated_at) VALUES (" + p(dialect, 1) + ", 1, 1, 1, " + p(dialect, 2) + ") ON CONFLICT(id) DO UPDATE SET sessions_revoked = 1, egress_quarantined = 1, old_writes_blocked = 1, activated_at = excluded.activated_at"
		_, err := tx.ExecContext(ctx, query, 1, now)
		return err
	})
}

// ReleaseRestoreQuarantine enables job egress after staging verification.
// Session revocation and the old-write fence remain in force.
func ReleaseRestoreQuarantine(ctx context.Context, db *sql.DB, dialect database.Dialect) error {
	if db == nil {
		return errors.New("restore database is required")
	}
	result, err := db.ExecContext(ctx, "UPDATE restore_controls SET egress_quarantined = 0 WHERE id = "+p(dialect, 1), 1)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected != 1 {
		return errors.New("restore controls are not active")
	}
	return nil
}

func RestoreWritesBlocked(ctx context.Context, db database.Executor, dialect database.Dialect) (bool, error) {
	var blocked int
	err := db.QueryRowContext(ctx, "SELECT old_writes_blocked FROM restore_controls WHERE id = "+p(dialect, 1), 1).Scan(&blocked)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return blocked == 1, nil
}

func p(dialect database.Dialect, position int) string {
	if dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
