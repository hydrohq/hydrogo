package security

import (
	"context"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/redis/go-redis/v9"
)

func TestRedisTenantAdmissionFailsClosedWithoutSharedStore(t *testing.T) {
	scope, _ := tenant.NewScope("tenant-a", "a")
	ctx := tenant.WithScope(context.Background(), scope)
	for _, admission := range []RedisTenantAdmission{
		{},
		{PerTenant: 1, LeaseTTL: time.Second},
		{Client: redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}), PerTenant: 1, LeaseTTL: time.Second},
	} {
		release, err := admission.Acquire(ctx)
		if release != nil || err != ErrTenantAdmissionDenied {
			t.Fatalf("admission without a usable shared store = release-present=%t err=%v", release != nil, err)
		}
	}
}
