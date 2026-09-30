package server

import "net/http"

// ConcurrencyLimitMiddleware caps the number of requests this process handles
// concurrently (ADR-0333). Once the cap is reached, new requests fail fast
// with 503 instead of queuing behind already-admitted work, so CPU already
// spent on in-flight requests is not starved by requests the process cannot
// serve promptly anyway. A non-positive limit disables the middleware with no
// added overhead, per the Compact-profile zero-footprint rule.
func ConcurrencyLimitMiddleware(limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			return nil
		}
		if limit <= 0 {
			return next
		}
		tokens := make(chan struct{}, limit)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
