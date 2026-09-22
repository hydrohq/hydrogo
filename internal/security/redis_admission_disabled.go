//go:build hydro_no_redis

package security

import (
	"context"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// RedisTenantAdmission preserves the server configuration contract in builds
// that omit Redis. It cannot grant a distributed lease.
type RedisTenantAdmission struct{}

func (RedisTenantAdmission) Acquire(context.Context) (func(), error) {
	return nil, ErrTenantAdmissionDenied
}

func (RedisTenantAdmission) Middleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := tenant.FromContext(r.Context()); err != nil {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "tenant capacity unavailable", http.StatusTooManyRequests)
	})
}
