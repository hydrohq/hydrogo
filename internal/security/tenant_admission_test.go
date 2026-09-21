package security

import (
	"context"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"testing"
	"time"
)

func TestTenantAdmissionSeparatesCapacityAndFailsClosed(t *testing.T) {
	a := &TenantAdmission{PerTenant: 1, MaxTenants: 2}
	if err := a.Acquire(context.Background()); err != ErrTenantAdmissionDenied {
		t.Fatalf("missing scope=%v", err)
	}
	one, _ := tenant.NewScope("a", "a")
	two, _ := tenant.NewScope("b", "b")
	three, _ := tenant.NewScope("c", "c")
	ctxA, ctxB, ctxC := tenant.WithScope(context.Background(), one), tenant.WithScope(context.Background(), two), tenant.WithScope(context.Background(), three)
	if err := a.Acquire(ctxA); err != nil {
		t.Fatal(err)
	}
	if err := a.Acquire(ctxB); err != nil {
		t.Fatal(err)
	}
	if err := a.Acquire(ctxC); err != ErrTenantAdmissionDenied {
		t.Fatalf("max tenants=%v", err)
	}
	a.Release(ctxA)
	ctx, cancel := context.WithTimeout(ctxA, 20*time.Millisecond)
	defer cancel()
	if err := a.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
	a.Release(ctx)
	a.Release(ctxB)
}
