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
	requests            atomic.Uint64
	inFlight            atomic.Int64
	responses           [6]atomic.Uint64
	latency             [6]atomic.Uint64
	latencyCount        atomic.Uint64
	latencyNanos        atomic.Uint64
	databaseStats       func() DatabaseStats
	cacheStats          func() CacheStats
	jobStats            func() JobStats
	workerProcessed     atomic.Uint64
	workerFailures      atomic.Uint64
	workerEnqueueErrors atomic.Uint64
	workerRetries       atomic.Uint64
	workerDeadLetters   atomic.Uint64
	workerConcurrency   atomic.Int64
}

type DatabaseStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUseConnections   int
	IdleConnections    int
	WaitCount          int64
	WaitDuration       time.Duration
}

type CacheStats struct {
	Hits        uint64
	Misses      uint64
	Loads       uint64
	LoadErrors  uint64
	Evictions   uint64
	Expirations uint64
}

type JobStats struct {
	Depth        int64
	Leased       int64
	DeadLettered int64
}

var latencyBuckets = [...]time.Duration{5 * time.Millisecond, 25 * time.Millisecond, 100 * time.Millisecond, 500 * time.Millisecond, time.Second, 5 * time.Second}

func NewMetrics() *Metrics { return &Metrics{} }

// SetDatabaseStatsProvider installs a process-local provider before the
// server starts. The provider must not expose user-controlled labels.
func (m *Metrics) SetDatabaseStatsProvider(provider func() DatabaseStats) {
	if m != nil {
		m.databaseStats = provider
	}
}

func (m *Metrics) SetCacheStatsProvider(provider func() CacheStats) {
	if m != nil {
		m.cacheStats = provider
	}
}

func (m *Metrics) SetJobStatsProvider(provider func() JobStats) {
	if m != nil {
		m.jobStats = provider
	}
}

func (m *Metrics) RecordWorkerProcessed() {
	if m != nil {
		m.workerProcessed.Add(1)
	}
}
func (m *Metrics) RecordWorkerFailure() {
	if m != nil {
		m.workerFailures.Add(1)
	}
}
func (m *Metrics) RecordWorkerEnqueueError() {
	if m != nil {
		m.workerEnqueueErrors.Add(1)
	}
}

func (m *Metrics) RecordWorkerRetry() {
	if m != nil {
		m.workerRetries.Add(1)
	}
}

func (m *Metrics) RecordWorkerDeadLetter() {
	if m != nil {
		m.workerDeadLetters.Add(1)
	}
}

func (m *Metrics) SetWorkerConcurrency(value int) {
	if m != nil && value > 0 {
		m.workerConcurrency.Store(int64(value))
	}
}

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
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
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
	if m.databaseStats != nil {
		stats := m.databaseStats()
		fmt.Fprintf(w, "hydro_database_pool_max_open_connections %d\n", stats.MaxOpenConnections)
		fmt.Fprintf(w, "hydro_database_pool_open_connections %d\n", stats.OpenConnections)
		fmt.Fprintf(w, "hydro_database_pool_in_use_connections %d\n", stats.InUseConnections)
		fmt.Fprintf(w, "hydro_database_pool_idle_connections %d\n", stats.IdleConnections)
		fmt.Fprintf(w, "hydro_database_pool_wait_count_total %d\n", stats.WaitCount)
		fmt.Fprintf(w, "hydro_database_pool_wait_duration_seconds_total %.9f\n", stats.WaitDuration.Seconds())
	}
	if m.cacheStats != nil {
		stats := m.cacheStats()
		fmt.Fprintf(w, "hydro_cache_hits_total %d\n", stats.Hits)
		fmt.Fprintf(w, "hydro_cache_misses_total %d\n", stats.Misses)
		fmt.Fprintf(w, "hydro_cache_loads_total %d\n", stats.Loads)
		fmt.Fprintf(w, "hydro_cache_load_errors_total %d\n", stats.LoadErrors)
		fmt.Fprintf(w, "hydro_cache_evictions_total %d\n", stats.Evictions)
		fmt.Fprintf(w, "hydro_cache_expirations_total %d\n", stats.Expirations)
	}
	if m.jobStats != nil {
		stats := m.jobStats()
		fmt.Fprintf(w, "hydro_jobs_depth %d\n", stats.Depth)
		fmt.Fprintf(w, "hydro_jobs_leased %d\n", stats.Leased)
		fmt.Fprintf(w, "hydro_jobs_dead_lettered %d\n", stats.DeadLettered)
	}
	fmt.Fprintf(w, "hydro_worker_processed_total %d\n", m.workerProcessed.Load())
	fmt.Fprintf(w, "hydro_worker_failures_total %d\n", m.workerFailures.Load())
	fmt.Fprintf(w, "hydro_worker_enqueue_errors_total %d\n", m.workerEnqueueErrors.Load())
	fmt.Fprintf(w, "hydro_worker_retries_total %d\n", m.workerRetries.Load())
	fmt.Fprintf(w, "hydro_worker_dead_letters_total %d\n", m.workerDeadLetters.Load())
	fmt.Fprintf(w, "hydro_worker_concurrency %d\n", m.workerConcurrency.Load())
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

func (r *responseRecorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
