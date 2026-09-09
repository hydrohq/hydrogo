package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type Store struct {
	Dialect database.Dialect
}

func (s Store) Record(ctx context.Context, tx database.Executor, id, action, targetID string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	var requestID any
	if value := security.RequestIDFromContext(ctx); value != "" {
		requestID = value
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO audit_events (id, action, target_id, request_id) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+")", id, action, targetID, requestID)
	return err
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
