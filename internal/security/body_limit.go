package security

import "net/http"

// DefaultRequestBodyLimit is used when a server embedding the core does not
// provide a profile-specific limit. A zero-value configuration must remain
// bounded rather than silently disabling request-size protection.
const DefaultRequestBodyLimit int64 = 1 << 20

// RequestBodyLimitMiddleware bounds general request bodies. Multipart media
// uploads can be excluded because they have their own profile-aware limit.
func RequestBodyLimitMiddleware(maxBytes int64, skip func(*http.Request) bool) func(http.Handler) http.Handler {
	if maxBytes <= 0 || maxBytes == 1<<63-1 {
		maxBytes = DefaultRequestBodyLimit
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip != nil && skip(r) {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > maxBytes {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
