package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type Store struct {
	Dialect database.Dialect
}

func (s Store) Enqueue(ctx context.Context, tx database.Executor, id, topic, aggregateID string, payloadVersion int) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO outbox_events (id, topic, aggregate_id, payload_version) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+", "+s.placeholder(3)+", "+s.placeholder(4)+")", id, topic, aggregateID, payloadVersion)
	return err
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
