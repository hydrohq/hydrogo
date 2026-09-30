package server

import "net/http"

// ConcurrencyLimitMiddleware caps the number of requests this process handles
// concurrently (ADR-0333). Once the cap is reached, new requests fail fast
// with 503 instead of queuing behind already-admitted work, so CPU already
// spent on in-flight requests is not starved by requests the process cannot
// serve promptly anyway. A non-positive limit disables the middleware with no
// added overhead, per the Compact-profile zero-footprint rule.
//
// bypass exempts requests an orchestrator or operator depends on during the
// exact overload this middleware reacts to (health/readiness probes, metrics
// scraping): shedding those would make a load balancer or Kubernetes kill the
// instance that is trying to protect itself, or blind an operator at the
// moment they most need visibility. A nil bypass exempts nothing.
func ConcurrencyLimitMiddleware(limit int, bypass func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			return nil
		}
		if limit <= 0 {
			return next
		}
		tokens := make(chan struct{}, limit)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if bypass != nil && bypass(r) {
				next.ServeHTTP(w, r)
				return
			}
			select {
			case tokens <- struct{}{}:
				defer func() { <-tokens }()
				next.ServeHTTP(w, r)
			default:
				w.Header().Set("Retry-After", "1")
				http.Error(w, "server is busy, please retry", http.StatusServiceUnavailable)
			}
		})
	}
}
