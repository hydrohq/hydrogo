// Package observability contains bounded, process-local operational signals.
package observability

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// Metrics is safe for concurrent HTTP handlers. It intentionally has no
// unbounded label cardinality: status is the only exported dimension.
type Metrics struct {
	requests  atomic.Uint64
	inFlight  atomic.Int64
	responses [6]atomic.Uint64
}

func NewMetrics() *Metrics { return &Metrics{} }

// Middleware records request count, in-flight requests and coarse response
// status counts without changing the request or response semantics.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requests.Add(1)
		m.inFlight.Add(1)
		defer m.inFlight.Add(-1)
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		bucket := recorder.status / 100
		if bucket >= 1 && bucket <= 5 {
			m.responses[bucket].Add(1)
		}
	})
}

// Handler returns Prometheus-compatible text with no user-controlled labels.
func (m *Metrics) Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "hydro_http_requests_total %d\n", m.requests.Load())
	fmt.Fprintf(w, "hydro_http_in_flight %d\n", m.inFlight.Load())
	for status := 1; status <= 5; status++ {
		fmt.Fprintf(w, "hydro_http_responses_total{class=\"%dxx\"} %d\n", status, m.responses[status].Load())
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
}

func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
