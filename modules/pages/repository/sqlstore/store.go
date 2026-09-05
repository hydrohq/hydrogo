package sqlstore

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type Store struct {
	Dialect database.Dialect
}

func (s Store) InsertDraft(ctx context.Context, tx database.Executor, id, title string) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO pages (id, title) VALUES ("+s.placeholder(1)+", "+s.placeholder(2)+")", id, title)
	return err
}

func (s Store) UpdateTitle(ctx context.Context, tx database.Executor, id string, expectedVersion int, title string) (bool, error) {
	result, err := tx.ExecContext(ctx,
		"UPDATE pages SET title = "+s.placeholder(1)+", version = version + 1 WHERE id = "+s.placeholder(2)+" AND version = "+s.placeholder(3),
		title, id, expectedVersion,
	)
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	return changed == 1, err
}

func (s Store) placeholder(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
