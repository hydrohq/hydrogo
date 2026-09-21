package security

import (
	"context"
	"os"
	"strings"
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

func TestRedisTenantAdmissionIntegrationSharesLeasesAcrossInstances(t *testing.T) {
	rawURL := strings.TrimSpace(os.Getenv("HYDRO_TEST_REDIS_URL"))
	if rawURL == "" {
		t.Skip("HYDRO_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	clientA, err := NewRedisClient(ctx, rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer clientA.Close()
	clientB, err := NewRedisClient(ctx, rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer clientB.Close()
	prefix := "hydro:test:tenant-admission:" + time.Now().UTC().Format("20060102150405.000000000") + ":"
	a := RedisTenantAdmission{Client: clientA, Prefix: prefix, PerTenant: 1, LeaseTTL: 2 * time.Second}
	b := RedisTenantAdmission{Client: clientB, Prefix: prefix, PerTenant: 1, LeaseTTL: 2 * time.Second}
	scopeA, _ := tenant.NewScope("redis-tenant-a", "a")
	scopeB, _ := tenant.NewScope("redis-tenant-b", "b")
	ctxA := tenant.WithScope(ctx, scopeA)
	ctxB := tenant.WithScope(ctx, scopeB)
	releaseA, err := a.Acquire(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseA()
	if release, err := b.Acquire(ctxA); release != nil || err != ErrTenantAdmissionDenied {
		t.Fatalf("shared tenant capacity was not enforced: release=%t err=%v", release != nil, err)
	}
	releaseB, err := b.Acquire(ctxB)
	if err != nil {
		t.Fatal(err)
	}
	releaseB()
	releaseA()
	if release, err := b.Acquire(ctxA); err != nil || release == nil {
		t.Fatalf("released shared tenant capacity was not recovered: release=%t err=%v", release != nil, err)
	} else {
		release()
	}
}
