// Package observability contains bounded, process-local operational signals.
package observability

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Metrics is safe for concurrent HTTP handlers. It intentionally has no
// unbounded label cardinality: status is the only exported dimension.
type Metrics struct {
	requests     atomic.Uint64
	inFlight     atomic.Int64
	responses    [6]atomic.Uint64
	latency      [6]atomic.Uint64
	latencyCount atomic.Uint64
	latencyNanos atomic.Uint64
}

var latencyBuckets = [...]time.Duration{5 * time.Millisecond, 25 * time.Millisecond, 100 * time.Millisecond, 500 * time.Millisecond, time.Second, 5 * time.Second}

func NewMetrics() *Metrics { return &Metrics{} }

// Middleware records request count, in-flight requests and coarse response
// status counts without changing the request or response semantics.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		m.requests.Add(1)
		m.inFlight.Add(1)
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			elapsed := time.Since(started)
			m.latencyCount.Add(1)
			m.latencyNanos.Add(uint64(elapsed.Nanoseconds()))
			for i, bucket := range latencyBuckets {
				if elapsed <= bucket {
					m.latency[i].Add(1)
				}
			}
			m.inFlight.Add(-1)
			bucket := recorder.status / 100
			if bucket >= 1 && bucket <= 5 {
				m.responses[bucket].Add(1)
			}
		}()
		next.ServeHTTP(recorder, r)
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
	for i, bucket := range latencyBuckets {
		fmt.Fprintf(w, "hydro_http_request_duration_seconds_bucket{le=\"%.3f\"} %d\n", bucket.Seconds(), m.latency[i].Load())
	}
	fmt.Fprintf(w, "hydro_http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", m.latencyCount.Load())
	fmt.Fprintf(w, "hydro_http_request_duration_seconds_count %d\n", m.latencyCount.Load())
	fmt.Fprintf(w, "hydro_http_request_duration_seconds_sum %.9f\n", float64(m.latencyNanos.Load())/float64(time.Second))
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
