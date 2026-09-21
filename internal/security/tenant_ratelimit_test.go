package security

import (
	"context"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestTenantScopedRateLimiterSeparatesBucketsAndFailsClosed(t *testing.T) {
	inner := NewAttemptLimiter(10, 1, time.Minute)
	limiter := TenantScopedRateLimiter{Inner: inner}
	a, _ := tenant.NewScope("tenant-a", "a")
	b, _ := tenant.NewScope("tenant-b", "b")
	if AllowForRequest(limiter, context.Background(), "login:ip") {
		t.Fatal("missing tenant scope was allowed")
	}
	if !AllowForRequest(limiter, tenant.WithScope(context.Background(), a), "login:ip") {
		t.Fatal("first tenant-a request was denied")
	}
	if AllowForRequest(limiter, tenant.WithScope(context.Background(), a), "login:ip") {
		t.Fatal("tenant-a limit was not enforced")
	}
	if !AllowForRequest(limiter, tenant.WithScope(context.Background(), b), "login:ip") {
		t.Fatal("tenant-b was affected by tenant-a bucket")
	}
}
