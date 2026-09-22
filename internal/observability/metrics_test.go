package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestMetricsExposeDeadLetterDepth(t *testing.T) {
	m := NewMetrics()
	m.SetWorkerConcurrency(4)
	m.RecordWorkerRetry()
	m.RecordWorkerDeadLetter()
	m.RecordReadinessFailure()
	m.SetReadinessState(true)
	m.RecordSessionCleanup(3)
	m.RecordSessionCleanupError()
	m.SetJobStatsProvider(func() JobStats {
		return JobStats{Depth: 2, Leased: 1, DeadLettered: 3}
	})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res := httptest.NewRecorder()
	m.Handler(res, req)
	if !strings.Contains(res.Body.String(), "hydro_jobs_dead_lettered 3\n") {
		t.Fatalf("metrics did not expose dead-letter count: %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "hydro_worker_concurrency 4\n") {
		t.Fatalf("metrics did not expose worker concurrency: %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "hydro_worker_retries_total 1\n") || !strings.Contains(res.Body.String(), "hydro_worker_dead_letters_total 1\n") {
		t.Fatalf("metrics did not expose worker outcomes: %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "hydro_readiness_failures_total 1\n") {
		t.Fatalf("metrics did not expose readiness failures: %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "hydro_readiness_state 1\n") {
		t.Fatalf("metrics did not expose readiness state: %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "hydro_session_cleanup_runs_total 1\n") || !strings.Contains(res.Body.String(), "hydro_session_cleanup_rows_total 3\n") || !strings.Contains(res.Body.String(), "hydro_session_cleanup_errors_total 1\n") {
		t.Fatalf("metrics did not expose session cleanup outcomes: %s", res.Body.String())
	}
}

func TestMetricsSnapshotIncludesRuntimeAndProviders(t *testing.T) {
	m := NewMetrics()
	m.SetReadinessState(true)
	m.SetWorkerConcurrency(3)
	m.SetDatabaseStatsProvider(func() DatabaseStats { return DatabaseStats{OpenConnections: 2, MaxOpenConnections: 8} })
	m.SetCacheStatsProvider(func() CacheStats { return CacheStats{Hits: 4, Misses: 1} })
	m.SetJobStatsProvider(func() JobStats { return JobStats{Depth: 5, Leased: 2} })
	s := m.Snapshot()
	if !s.ReadinessState || !s.HasDatabase || !s.HasCache || !s.HasJobs {
		t.Fatalf("snapshot provider/readiness state = %#v", s)
	}
	if s.CPUCount < 1 || s.GOMAXPROCS < 1 || s.Goroutines < 1 {
		t.Fatalf("snapshot runtime values = %#v", s)
	}
	if s.Database.OpenConnections != 2 || s.Cache.Hits != 4 || s.Jobs.Depth != 5 || s.WorkerConcurrency != 3 {
		t.Fatalf("snapshot provider values = %#v", s)
	}
}

func TestMetricsSkipStaticAssetPaths(t *testing.T) {
	m := NewMetrics()
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/public/news/tr/html" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	for _, path := range []string{"/vendor/quill/quill.js", "/vendor/quill/quill.snow.css", "/theme/static/theme.css", "/public/news/tr/html"} {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/public/news/tr/html", nil))
	stats := m.PathStats()
	if len(stats) != 1 || stats[0].Path != "/public/news/tr/html" || stats[0].Views != 2 || stats[0].Requests != 2 {
		t.Fatalf("path stats = %#v, want two public requests/views", stats)
	}
}

func TestMetricsEmptyState(t *testing.T) {
	m := NewMetrics()
	if m.TotalRequests() != 0 {
		t.Errorf("TotalRequests = %d, want 0", m.TotalRequests())
	}
	if m.TotalViews() != 0 {
		t.Errorf("TotalViews = %d, want 0", m.TotalViews())
	}
	if m.TotalErrors() != 0 {
		t.Errorf("TotalErrors = %d, want 0", m.TotalErrors())
	}
	if m.AvgLatencyMS() != 0 {
		t.Errorf("AvgLatencyMS = %d, want 0", m.AvgLatencyMS())
	}
	if m.UniqueVisitorsCount() != 0 {
		t.Errorf("UniqueVisitorsCount = %d, want 0", m.UniqueVisitorsCount())
	}
	if len(m.PathStats()) != 0 {
		t.Errorf("PathStats length = %d, want 0", len(m.PathStats()))
	}
	if len(m.TopPages(5)) != 0 {
		t.Errorf("TopPages length = %d, want 0", len(m.TopPages(5)))
	}
	if len(m.TopErrors(5)) != 0 {
		t.Errorf("TopErrors length = %d, want 0", len(m.TopErrors(5)))
	}
}

func TestMetricsPublicTelemetryDoesNotTrackVisitors(t *testing.T) {
	m := NewMetrics()
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/public/news/tr/html", nil)
		req.RemoteAddr = "192.168.1.50:1234"
		handler.ServeHTTP(rec, req)
	}

	stats := m.PathStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 path stats entry, got %d", len(stats))
	}
	if stats[0].Views != 5 {
		t.Errorf("expected one operational view per successful public request, got %d", stats[0].Views)
	}
	if stats[0].Requests != 5 {
		t.Errorf("expected 5 total requests, got %d", stats[0].Requests)
	}
	if m.UniqueVisitorsCount() != 0 {
		t.Errorf("operational telemetry must not count unique visitors, got %d", m.UniqueVisitorsCount())
	}
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/public/news/tr/html", nil)
	req2.RemoteAddr = "10.0.0.99:5678"
	handler.ServeHTTP(rec2, req2)

	if m.UniqueVisitorsCount() != 0 {
		t.Errorf("operational telemetry must not count unique visitors, got %d", m.UniqueVisitorsCount())
	}
	if m.TotalViews() != 6 {
		t.Errorf("expected 6 public requests/views, got %d", m.TotalViews())
	}
}

func TestMetricsProcessRestartResetsInMemoryCounters(t *testing.T) {
	// Simulate Process 1
	p1 := NewMetrics()
	h1 := p1.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h1.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/public/pages/tr/demo/html", nil))
	h1.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/public/pages/tr/demo/html", nil))

	if p1.TotalRequests() != 2 || p1.TotalViews() != 2 {
		t.Fatalf("process 1 state unexpected: req=%d, views=%d", p1.TotalRequests(), p1.TotalViews())
	}

	// Process Restart (new in-memory metrics instance)
	p2 := NewMetrics()
	if p2.TotalRequests() != 0 || p2.TotalViews() != 0 || p2.UniqueVisitorsCount() != 0 {
		t.Errorf("restarted process must start with 0 metrics (in-memory lifetime); got req=%d, views=%d", p2.TotalRequests(), p2.TotalViews())
	}
	if p2.StartedAt().Before(p1.StartedAt()) {
		t.Errorf("restarted process startedAt must be newer or equal")
	}
}

func TestMetricsErrorTrackingAndTopErrors(t *testing.T) {
	m := NewMetrics()
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/error":
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/v1/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/api/v1/ok":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))

	// Generate 3 server errors on /api/v1/error
	for i := 0; i < 3; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/error", nil))
	}
	// Generate 1 client error on /api/v1/notfound
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/notfound", nil))
	// Generate 5 success requests on /api/v1/ok
	for i := 0; i < 5; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/ok", nil))
	}

	if m.TotalErrors() != 4 {
		t.Errorf("TotalErrors = %d, want 4", m.TotalErrors())
	}
	topErrors := m.TopErrors(5)
	if len(topErrors) != 2 {
		t.Fatalf("expected 2 error routes, got %d", len(topErrors))
	}
	if topErrors[0].Path != "/api/v1/error" || topErrors[0].Errors != 3 {
		t.Errorf("top error = %+v, want /api/v1/error with 3 errors", topErrors[0])
	}
	if topErrors[1].Path != "/api/v1/notfound" || topErrors[1].Errors != 1 {
		t.Errorf("second error = %+v, want /api/v1/notfound with 1 error", topErrors[1])
	}
}

func TestMetricsExposeBoundedEndpointPercentilesBytesAndStatuses(t *testing.T) {
	m := NewMetrics()
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/project/p-1" {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("limited"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/project/p-1", strings.NewReader("in"))
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	stats := m.PathStats()
	if len(stats) != 1 {
		t.Fatalf("stats = %#v", stats)
	}
	got := stats[0]
	if got.Path != "/api/v1/project/{id}" {
		t.Fatalf("route was not normalized: %#v", got)
	}
	if got.Requests != 3 || got.Status4xx != 3 || got.RateLimited != 3 {
		t.Fatalf("endpoint counters = %#v", got)
	}
	if got.BytesIn != uint64(len("in"))*3 || got.BytesOut != uint64(len("limited"))*3 {
		t.Fatalf("endpoint bytes = %#v", got)
	}
}

func TestMetricsExposeTenantAndModuleDimensions(t *testing.T) {
	m := NewMetrics()
	scope, _ := tenant.NewScope("tenant-a", "alpha")
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p-1", nil).WithContext(tenant.WithScope(context.Background(), scope))
	h.ServeHTTP(httptest.NewRecorder(), req)
	dims := m.DimensionStats()
	if len(dims) != 1 || dims[0].TenantID != "tenant-a" || dims[0].Module != "projects" || dims[0].Requests != 1 {
		t.Fatalf("unexpected dimensions: %#v", dims)
	}
}
