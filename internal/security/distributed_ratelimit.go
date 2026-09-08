package security

import "context"

// DistributedAttemptStore is the minimal shared-counter contract required by
// multi-instance authentication limiting. Implementations own atomicity and
// TTL in the shared store (for example Redis); the application never assumes
// a local counter is globally authoritative.
type DistributedAttemptStore interface {
	Allow(context.Context, string) (bool, error)
}

// DistributedLimiter fails closed when the shared store is unavailable. This
// prevents a backend outage from silently removing login/MFA protection.
type DistributedLimiter struct{ Store DistributedAttemptStore }

func (l DistributedLimiter) Allow(ctx context.Context, key string) bool {
	if l.Store == nil || ctx.Err() != nil {
		return false
	}
	ok, err := l.Store.Allow(ctx, key)
	return err == nil && ok
}
