package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type Store struct {
	Dialect database.Dialect
}

func (s Store) Record(ctx context.Context, tx database.Executor, id, action, targetID string) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO audit_events (id, action, target_id) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+")", id, action, targetID)
	return err
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
