package security

import (
	"context"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"net/http"
	"net/http/httptest"
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

func TestTenantAdmissionMiddlewareReturns429AtCapacity(t *testing.T) {
	a := &TenantAdmission{PerTenant: 1}
	scope, _ := tenant.NewScope("tenant-a", "a")
	ctx := tenant.WithScope(context.Background(), scope)
	if err := a.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d", res.Code)
	}
	a.Release(ctx)
}

func TestTenantAdmissionCancellationRecoversCapacity(t *testing.T) {
	a := &TenantAdmission{PerTenant: 1}
	scope, _ := tenant.NewScope("tenant-recovery", "recovery")
	ctx := tenant.WithScope(context.Background(), scope)
	if err := a.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()
	if err := a.Acquire(canceled); err == nil {
		t.Fatal("expected blocked admission to honor cancellation")
	}
	a.Release(ctx)
	if err := a.Acquire(ctx); err != nil {
		t.Fatalf("capacity did not recover after cancellation/release: %v", err)
	}
	a.Release(ctx)
}
