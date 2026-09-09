package erasure

import "context"

// GC coordinates storage deletion with a final pin check. The storage
// provider must make the pin lookup and delete operation idempotent; a newly
// pinned object is never deleted after the final check in this process.
type GC struct {
	Pinned func(context.Context, string) (bool, error)
	Delete func(context.Context, string) error
}

func (g GC) Collect(ctx context.Context, keys []string) ([]string, error) {
	if g.Pinned == nil || g.Delete == nil {
		return nil, ErrInvalidIntent
	}
	if ctx == nil {
		ctx = context.Background()
	}
	removed := make([]string, 0, len(keys))
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		pinned, err := g.Pinned(ctx, key)
		if err != nil {
			return removed, err
		}
		if pinned {
			continue
		}
		if err := g.Delete(ctx, key); err != nil {
			return removed, err
		}
		removed = append(removed, key)
	}
	return removed, nil
}
