package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/redis/go-redis/v9"
)

// RedisTenantAdmission is an opt-in distributed concurrency lease. Each
// tenant has a sorted-set of unexpired request leases, so multiple HydroGo
// instances share one noisy-neighbor budget without keeping process state.
type RedisTenantAdmission struct {
	Client    *redis.Client
	Prefix    string
	PerTenant int
	LeaseTTL  time.Duration
}

func (a RedisTenantAdmission) Acquire(ctx context.Context) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	scope, err := tenant.Require(ctx)
	if err != nil || a.Client == nil || a.PerTenant <= 0 || a.LeaseTTL <= 0 {
		return nil, ErrTenantAdmissionDenied
	}
	var tokenBytes [16]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return nil, ErrTenantAdmissionDenied
	}
	token := hex.EncodeToString(tokenBytes[:])
	key := a.Prefix + "tenant-admission:" + scope.TenantID
	now := time.Now().UnixMilli()
	expires := now + a.LeaseTTL.Milliseconds()
	const script = `redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1]); if redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[3]) then return 0 end; redis.call('ZADD', KEYS[1], ARGV[2], ARGV[4]); redis.call('PEXPIRE', KEYS[1], ARGV[5]); return 1`
	result, err := a.Client.Eval(ctx, script, []string{key}, now, expires, a.PerTenant, token, a.LeaseTTL.Milliseconds()*2).Int()
	if err != nil || result != 1 {
		return nil, ErrTenantAdmissionDenied
	}
	var once sync.Once
	release := func() {
		once.Do(func() { _ = a.Client.ZRem(context.Background(), key, token).Err() })
	}
	return release, nil
}

func (a RedisTenantAdmission) Middleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := tenant.FromContext(r.Context()); err != nil {
			next.ServeHTTP(w, r)
			return
		}
		release, err := a.Acquire(r.Context())
		if err != nil {
			http.Error(w, "tenant capacity unavailable", http.StatusTooManyRequests)
			return
		}
		defer release()
		next.ServeHTTP(w, r)
	})
}
