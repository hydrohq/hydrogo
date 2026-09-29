// Package observability contains bounded, process-local operational signals.
package observability

import (
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// Metrics is safe for concurrent HTTP handlers. It intentionally has no
// unbounded label cardinality: status is the only exported dimension.
type Metrics struct {
	startedAt           time.Time
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
	readinessFailures   atomic.Uint64
	readinessState      atomic.Int32
	pathMu              sync.RWMutex
	paths               map[string]PathStats
	pathSamples         map[string][]uint64
	dimensionStats      map[string]DimensionStats
	sessionCleanupRuns  atomic.Uint64
	sessionCleanupRows  atomic.Uint64
	sessionCleanupError atomic.Uint64
}

type PathStats struct {
	Path                                       string
	Requests, Views, Errors, TotalMS           uint64
	BytesIn, BytesOut                          uint64
	Status2xx, Status3xx, Status4xx, Status5xx uint64
	Timeouts, RateLimited                      uint64
}

type PathStatsSnapshot struct {
	Path                                       string
	Requests, Views, Errors, AvgMS             uint64
	P50MS, P95MS, P99MS                        uint64
	BytesIn, BytesOut                          uint64
	Status2xx, Status3xx, Status4xx, Status5xx uint64
	Timeouts, RateLimited                      uint64
}

type DimensionStats struct {
	TenantID, Module          string
	Requests, Errors, TotalMS uint64
}
type DimensionStatsSnapshot struct {
	TenantID, Module        string
	Requests, Errors, AvgMS uint64
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

// RuntimeSnapshot is a bounded, point-in-time view for the authenticated
// Admin Studio resource page. It intentionally contains process/runtime and
// already-registered provider statistics only; it does not invent host-level
// CPU, disk, or historical values that the process cannot measure.
type RuntimeSnapshot struct {
	StartedAt         time.Time
	Uptime            time.Duration
	Requests          uint64
	InFlight          int64
	Views             uint64
	Errors            uint64
	AvgLatencyMS      uint64
	UniqueVisitors    int
	Goroutines        int
	CPUCount          int
	GOMAXPROCS        int
	HeapAllocBytes    uint64
	HeapInUseBytes    uint64
	HeapSysBytes      uint64
	HeapObjects       uint64
	Database          DatabaseStats
	Cache             CacheStats
	Jobs              JobStats
	HasDatabase       bool
	HasCache          bool
	HasJobs           bool
	WorkerProcessed   uint64
	WorkerFailures    uint64
	WorkerRetries     uint64
	DeadLetters       uint64
	WorkerConcurrency int64
	ReadinessState    bool
	ReadinessFails    uint64
}

var latencyBuckets = [...]time.Duration{5 * time.Millisecond, 25 * time.Millisecond, 100 * time.Millisecond, 500 * time.Millisecond, time.Second, 5 * time.Second}

func NewMetrics() *Metrics {
	m := &Metrics{
		startedAt:   time.Now().UTC(),
		paths:       make(map[string]PathStats),
		pathSamples: make(map[string][]uint64),
	}
	return m
}

func (m *Metrics) PathStats() []PathStatsSnapshot {
	if m == nil {
		return nil
	}
	m.pathMu.RLock()
	items := make([]PathStatsSnapshot, 0, len(m.paths))
	for path, stats := range m.paths {
		avg := uint64(0)
		if stats.Requests > 0 {
			avg = stats.TotalMS / stats.Requests
		}
		items = append(items, PathStatsSnapshot{
			Path: path, Requests: stats.Requests, Views: stats.Views, Errors: stats.Errors,
			AvgMS: avg, P50MS: percentile(m.pathSamples[path], 50), P95MS: percentile(m.pathSamples[path], 95), P99MS: percentile(m.pathSamples[path], 99),
			BytesIn: stats.BytesIn, BytesOut: stats.BytesOut, Status2xx: stats.Status2xx, Status3xx: stats.Status3xx,
			Status4xx: stats.Status4xx, Status5xx: stats.Status5xx, Timeouts: stats.Timeouts, RateLimited: stats.RateLimited,
		})
	}
	m.pathMu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Requests > items[j].Requests })
	return items
}

func (m *Metrics) StartedAt() time.Time {
	if m == nil {
		return time.Time{}
	}
	return m.startedAt
}

func (m *Metrics) TotalRequests() uint64 {
	if m == nil {
		return 0
	}
	return m.requests.Load()
}

func (m *Metrics) InFlight() int64 {
	if m == nil {
		return 0
	}
	return m.inFlight.Load()
}

func (m *Metrics) TotalErrors() uint64 {
	if m == nil {
		return 0
	}
	return m.responses[4].Load() + m.responses[5].Load()
}

func (m *Metrics) TotalViews() uint64 {
	if m == nil {
		return 0
	}
	m.pathMu.RLock()
	defer m.pathMu.RUnlock()
	var total uint64
	for _, s := range m.paths {
		total += s.Views
	}
	return total
}

func (m *Metrics) UniqueVisitorsCount() int {
	// Unique visitor measurement is product analytics, not operational
	// telemetry. It remains unavailable until the optional, consent-gated
	// analytics module is enabled.
	return 0
}

func (m *Metrics) AvgLatencyMS() uint64 {
	if m == nil {
		return 0
	}
	cnt := m.latencyCount.Load()
	if cnt == 0 {
		return 0
	}
	return (m.latencyNanos.Load() / cnt) / 1_000_000
}

// Snapshot returns only bounded, current process data suitable for an admin
// view. Provider callbacks are expected to be short and already use their own
// timeout/budget where a database query is involved.
func (m *Metrics) Snapshot() RuntimeSnapshot {
	if m == nil {
		return RuntimeSnapshot{}
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	now := time.Now().UTC()
	snapshot := RuntimeSnapshot{
		StartedAt:         m.startedAt,
		Uptime:            now.Sub(m.startedAt),
		Requests:          m.TotalRequests(),
		InFlight:          m.InFlight(),
		Views:             m.TotalViews(),
		Errors:            m.TotalErrors(),
		AvgLatencyMS:      m.AvgLatencyMS(),
		UniqueVisitors:    m.UniqueVisitorsCount(),
		Goroutines:        runtime.NumGoroutine(),
		CPUCount:          runtime.NumCPU(),
		GOMAXPROCS:        runtime.GOMAXPROCS(0),
		HeapAllocBytes:    mem.HeapAlloc,
		HeapInUseBytes:    mem.HeapInuse,
		HeapSysBytes:      mem.HeapSys,
		HeapObjects:       mem.HeapObjects,
		WorkerProcessed:   m.workerProcessed.Load(),
		WorkerFailures:    m.workerFailures.Load(),
		WorkerRetries:     m.workerRetries.Load(),
		DeadLetters:       m.workerDeadLetters.Load(),
		WorkerConcurrency: m.workerConcurrency.Load(),
		ReadinessState:    m.readinessState.Load() == 1,
		ReadinessFails:    m.readinessFailures.Load(),
	}
	if m.databaseStats != nil {
		snapshot.Database = m.databaseStats()
		snapshot.HasDatabase = true
	}
	if m.cacheStats != nil {
		snapshot.Cache = m.cacheStats()
		snapshot.HasCache = true
	}
	if m.jobStats != nil {
		snapshot.Jobs = m.jobStats()
		snapshot.HasJobs = true
	}
	return snapshot
}

func (m *Metrics) TopPages(limit int) []PathStatsSnapshot {
	stats := m.PathStats()
	sort.Slice(stats, func(i, j int) bool { return stats[i].Views > stats[j].Views })
	if limit > 0 && len(stats) > limit {
		return stats[:limit]
	}
	return stats
}

func (m *Metrics) TopErrors(limit int) []PathStatsSnapshot {
	stats := m.PathStats()
	filtered := make([]PathStatsSnapshot, 0, len(stats))
	for _, s := range stats {
		if s.Errors > 0 {
			filtered = append(filtered, s)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Errors > filtered[j].Errors })
	if limit > 0 && len(filtered) > limit {
		return filtered[:limit]
	}
	return filtered
}

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

func (m *Metrics) RecordSessionCleanup(deleted int64) {
	if m != nil {
		m.sessionCleanupRuns.Add(1)
		if deleted > 0 {
			m.sessionCleanupRows.Add(uint64(deleted))
		}
	}
}

func (m *Metrics) RecordSessionCleanupError() {
	if m != nil {
		m.sessionCleanupError.Add(1)
	}
}

func (m *Metrics) SetWorkerConcurrency(value int) {
	if m != nil && value > 0 {
		m.workerConcurrency.Store(int64(value))
	}
}

// RecordReadinessFailure records a failed readiness probe without exposing
// the underlying dependency error or any user-controlled labels.
func (m *Metrics) RecordReadinessFailure() {
	if m != nil {
		m.readinessFailures.Add(1)
	}
}

// SetReadinessState records the latest readiness result as a bounded gauge:
// 1 means ready and 0 means not ready. A new Metrics instance starts not
// ready until the first probe succeeds.
func (m *Metrics) SetReadinessState(ready bool) {
	if m == nil {
		return
	}
	if ready {
		m.readinessState.Store(1)
		return
	}
	m.readinessState.Store(0)
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
		if r.ContentLength > 0 {
			recorder.requestBytes = uint64(r.ContentLength)
		}
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
			m.recordPath(r, recorder.status, elapsed, recorder.bytes, recorder.requestBytes)
		}()
		next.ServeHTTP(recorder, r)
	})
}

func (m *Metrics) recordPath(r *http.Request, status int, elapsed time.Duration, bytesOut, bytesIn uint64) {
	if m == nil || r == nil || len(r.URL.Path) == 0 || len(r.URL.Path) > 128 || !trackablePath(r.URL.Path) {
		return
	}
	path := r.URL.Path
	if r.Pattern != "" {
		path = r.Pattern
	} else {
		path = normalizeRoute(r.URL.Path)
	}
	m.pathMu.Lock()
	if m.paths == nil {
		m.paths = make(map[string]PathStats)
	}
	if m.pathSamples == nil {
		m.pathSamples = make(map[string][]uint64)
	}
	stats, ok := m.paths[path]
	if !ok && len(m.paths) >= 64 {
		m.pathMu.Unlock()
		return
	}
	stats.Path = path
	stats.Requests++
	stats.TotalMS += uint64(elapsed.Milliseconds())
	stats.BytesIn += bytesIn
	stats.BytesOut += bytesOut
	switch status / 100 {
	case 2:
		stats.Status2xx++
	case 3:
		stats.Status3xx++
	case 4:
		stats.Status4xx++
	case 5:
		stats.Status5xx++
	}
	if status == http.StatusRequestTimeout || status == http.StatusGatewayTimeout {
		stats.Timeouts++
	}
	if status == http.StatusTooManyRequests {
		stats.RateLimited++
	}
	if status >= http.StatusBadRequest {
		stats.Errors++
	}
	if r.Method == http.MethodGet && (strings.HasPrefix(path, "/public/") || strings.HasPrefix(path, "/shop")) && status < http.StatusBadRequest {
		stats.Views++
	}
	m.paths[path] = stats
	samples := m.pathSamples[path]
	latencyMS := uint64(elapsed.Milliseconds())
	if len(samples) < 128 {
		samples = append(samples, latencyMS)
	} else {
		samples[(stats.Requests-1)%128] = latencyMS
	}
	m.pathSamples[path] = samples
	key := dimensionKey(r, path)
	if m.dimensionStats == nil {
		m.dimensionStats = make(map[string]DimensionStats)
	}
	if dim, exists := m.dimensionStats[key]; exists || len(m.dimensionStats) < 128 {
		if !exists {
			dim = DimensionStats{TenantID: dimensionTenant(r), Module: routeModule(path)}
		}
		dim.Requests++
		dim.TotalMS += uint64(elapsed.Milliseconds())
		if status >= 400 {
			dim.Errors++
		}
		m.dimensionStats[key] = dim
	}
	m.pathMu.Unlock()
}

func dimensionTenant(r *http.Request) string {
	if r != nil {
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			return scope.TenantID
		}
	}
	return "default"
}
func routeModule(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		if parts[0] == "api" && len(parts) >= 3 {
			return parts[2]
		}
		return parts[1]
	}
	return "root"
}

// RouteModule returns the bounded module label shared by logs and metrics.
func RouteModule(path string) string { return routeModule(path) }
func dimensionKey(r *http.Request, path string) string {
	return dimensionTenant(r) + "\x00" + routeModule(path)
}

func (m *Metrics) DimensionStats() []DimensionStatsSnapshot {
	if m == nil {
		return nil
	}
	m.pathMu.RLock()
	defer m.pathMu.RUnlock()
	out := make([]DimensionStatsSnapshot, 0, len(m.dimensionStats))
	for _, dim := range m.dimensionStats {
		avg := uint64(0)
		if dim.Requests > 0 {
			avg = dim.TotalMS / dim.Requests
		}
		out = append(out, DimensionStatsSnapshot{TenantID: dim.TenantID, Module: dim.Module, Requests: dim.Requests, Errors: dim.Errors, AvgMS: avg})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TenantID == out[j].TenantID {
			return out[i].Module < out[j].Module
		}
		return out[i].TenantID < out[j].TenantID
	})
	return out
}

// normalizeRoute keeps endpoint metrics useful without allowing customer IDs,
// slugs, or arbitrary path segments to create unbounded label cardinality.
// Public content paths intentionally remain content paths for the existing
// view report; API resource IDs are normalized after the resource segment.
func normalizeRoute(path string) string {
	if !strings.HasPrefix(path, "/api/") {
		return path
	}
	parts := strings.Split(path, "/")
	start := 3 // /api/<resource>/<id>
	if len(parts) > 2 && parts[2] == "v1" {
		start = 4 // /api/v1/<resource>/<id>
	}
	if len(parts) <= start {
		return path
	}
	for i := start; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

// NormalizeRoute exposes the bounded route label used by metrics and logs.
func NormalizeRoute(path string) string { return normalizeRoute(path) }

func percentile(samples []uint64, p int) uint64 {
	if len(samples) == 0 {
		return 0
	}
	copySamples := append([]uint64(nil), samples...)
	sort.Slice(copySamples, func(i, j int) bool { return copySamples[i] < copySamples[j] })
	index := (p*len(copySamples) + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(copySamples) {
		index = len(copySamples)
	}
	return copySamples[index-1]
}

func trackablePath(path string) bool {
	for _, prefix := range []string{"/vendor/", "/theme/", "/media/", "/favicon"} {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return true
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
	fmt.Fprintf(w, "hydro_readiness_failures_total %d\n", m.readinessFailures.Load())
	fmt.Fprintf(w, "hydro_readiness_state %d\n", m.readinessState.Load())
	fmt.Fprintf(w, "hydro_session_cleanup_runs_total %d\n", m.sessionCleanupRuns.Load())
	fmt.Fprintf(w, "hydro_session_cleanup_rows_total %d\n", m.sessionCleanupRows.Load())
	fmt.Fprintf(w, "hydro_session_cleanup_errors_total %d\n", m.sessionCleanupError.Load())
}

type responseRecorder struct {
	http.ResponseWriter
	status       int
	wroteHeader  bool
	bytes        uint64
	requestBytes uint64
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
	n, err := r.ResponseWriter.Write(body)
	r.bytes += uint64(n)
	return n, err
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
