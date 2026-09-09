package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

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
	if guards.IsNil(l.Store) || ctx.Err() != nil || key == "" {
		return false
	}
	// Keep the shared store from receiving raw IP addresses, email addresses,
	// or composite authentication keys. The digest is fixed-size and opaque,
	// while still preserving stable bucket identity across instances.
	digest := sha256.Sum256([]byte(key))
	opaqKey := hex.EncodeToString(digest[:])
	ok, err := l.Store.Allow(ctx, opaqKey)
	return err == nil && ok
}
