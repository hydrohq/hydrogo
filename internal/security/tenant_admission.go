package security

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

var ErrTenantAdmissionDenied = errors.New("tenant admission denied")

// TenantAdmission bounds concurrent work per tenant. It is deliberately
// process-local; distributed deployments must pair it with a shared gateway
// or queue quota rather than treating this as a global limit.
type TenantAdmission struct {
	PerTenant  int
	MaxTenants int
	mu         sync.Mutex
	entries    map[string]chan struct{}
}

func (a *TenantAdmission) Acquire(ctx context.Context) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return ErrTenantAdmissionDenied
	}
	if a == nil || a.PerTenant <= 0 {
		return ErrTenantAdmissionDenied
	}
	a.mu.Lock()
	if a.entries == nil {
		a.entries = make(map[string]chan struct{})
	}
	if a.MaxTenants <= 0 {
		a.MaxTenants = 1024
	}
	gate, ok := a.entries[scope.TenantID]
	if !ok {
		if len(a.entries) >= a.MaxTenants {
			a.mu.Unlock()
			return ErrTenantAdmissionDenied
		}
		gate = make(chan struct{}, a.PerTenant)
		a.entries[scope.TenantID] = gate
	}
	a.mu.Unlock()
	select {
	case gate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *TenantAdmission) tryAcquire(ctx context.Context) error {
	scope, err := tenant.Require(ctx)
	if err != nil {
		return ErrTenantAdmissionDenied
	}
	if a == nil || a.PerTenant <= 0 {
		return ErrTenantAdmissionDenied
	}
	a.mu.Lock()
	if a.entries == nil {
		a.entries = make(map[string]chan struct{})
	}
	if a.MaxTenants <= 0 {
		a.MaxTenants = 1024
	}
	gate, ok := a.entries[scope.TenantID]
	if !ok {
		if len(a.entries) >= a.MaxTenants {
			a.mu.Unlock()
			return ErrTenantAdmissionDenied
		}
		gate = make(chan struct{}, a.PerTenant)
		a.entries[scope.TenantID] = gate
	}
	a.mu.Unlock()
	select {
	case gate <- struct{}{}:
		return nil
	default:
		return ErrTenantAdmissionDenied
	}
}

func (a *TenantAdmission) Release(ctx context.Context) {
	scope, err := tenant.FromContext(ctx)
	if err != nil || a == nil {
		return
	}
	a.mu.Lock()
	gate := a.entries[scope.TenantID]
	a.mu.Unlock()
	if gate == nil {
		return
	}
	select {
	case <-gate:
	default:
	}
}

// Middleware applies admission only after a tenant resolver has established a
// scope. Single-tenant/public requests without a scope retain existing
// behavior; shared-tenant requests are rejected with 429 when their bounded
// concurrency budget is exhausted.
func (a *TenantAdmission) Middleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := tenant.FromContext(r.Context()); err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if err := a.tryAcquire(r.Context()); err != nil {
			http.Error(w, "tenant capacity unavailable", http.StatusTooManyRequests)
			return
		}
		defer a.Release(r.Context())
		next.ServeHTTP(w, r)
	})
}
