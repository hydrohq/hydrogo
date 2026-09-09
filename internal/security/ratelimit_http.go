package security

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

// WriteRateLimitedResponse tells clients when a bounded local limiter window
// is expected to become eligible again. The duration is rounded up so clients
// never retry before the advertised window has elapsed.
func WriteRateLimitedResponse(w http.ResponseWriter, message string, retryAfter time.Duration) {
	seconds := int64(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
	http.Error(w, message, http.StatusTooManyRequests)
}
