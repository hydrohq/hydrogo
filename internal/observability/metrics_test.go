package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsExposeDeadLetterDepth(t *testing.T) {
	m := NewMetrics()
	m.SetWorkerConcurrency(4)
	m.RecordWorkerRetry()
	m.RecordWorkerDeadLetter()
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
	if !strings.Contains(res.Body.String(), "hydro_session_cleanup_runs_total 1\n") || !strings.Contains(res.Body.String(), "hydro_session_cleanup_rows_total 3\n") || !strings.Contains(res.Body.String(), "hydro_session_cleanup_errors_total 1\n") {
		t.Fatalf("metrics did not expose session cleanup outcomes: %s", res.Body.String())
	}
}
