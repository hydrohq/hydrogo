package sqlstore

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// InTransaction implements collections.Transactional. Inside an open
// transaction it simply runs fn, so calls nest.
func (s Store) InTransaction(ctx context.Context, fn func(collections.Repository) error) error {
	return s.within(ctx, func(tx Store) error { return fn(tx) })
}
