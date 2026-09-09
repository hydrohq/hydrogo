package erasure

import (
	"context"
	"errors"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// SQLAdapter applies the stable, database-side part of an erasure intent.
// Storage garbage collection remains a separate idempotent operation keyed by
// the media storage key; deleting metadata never exposes a deleted asset.
type SQLAdapter struct {
	Dialect database.Dialect
}

func (a SQLAdapter) Apply(ctx context.Context, db database.Executor, intent Intent) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if a.Dialect != database.SQLite && a.Dialect != database.Postgres {
		return fmt.Errorf("unsupported database dialect %q", a.Dialect)
	}
	if err := database.RequireExecutor(db); err != nil {
		return errors.New("erasure database is required")
	}
	if intent.EditionID == "" || intent.TargetID == "" {
		return ErrInvalidIntent
	}
	switch intent.Kind {
	case "erase.page":
		if _, err := db.ExecContext(ctx, "DELETE FROM page_revisions WHERE page_id = "+a.placeholder(1), intent.TargetID); err != nil {
			return err
		}
		_, err := db.ExecContext(ctx, "DELETE FROM pages WHERE id = "+a.placeholder(1), intent.TargetID)
		return err
	case "erase.media":
		_, err := db.ExecContext(ctx, "DELETE FROM media_assets WHERE id = "+a.placeholder(1), intent.TargetID)
		return err
	default:
		return fmt.Errorf("unsupported erasure intent kind %q", intent.Kind)
	}
}

func (a SQLAdapter) placeholder(position int) string {
	if a.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
