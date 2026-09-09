package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

func TestOperationalRoutesAreExplicit(t *testing.T) {
	handler := NewHandler(Config{})

	for _, route := range []struct {
		path string
		want int
	}{
		{path: "/healthz", want: http.StatusOK},
		{path: "/readyz", want: http.StatusOK},
		{path: "/unregistered", want: http.StatusNotFound},
	} {
		req := httptest.NewRequest(http.MethodGet, route.path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if got := res.Code; got != route.want {
			t.Errorf("GET %s status = %d, want %d", route.path, got, route.want)
		}
		if res.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("GET %s missing Content-Security-Policy header", route.path)
		}
		requestID := res.Header().Get("X-Request-ID")
		if len(requestID) != 32 {
			t.Errorf("GET %s request id length = %d, want 32", route.path, len(requestID))
		}
	}
}

func TestRequestIDIsFreshAndClientValueIsNotTrusted(t *testing.T) {
	handler := NewHandler(Config{})
	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	firstRequest.Header.Set("X-Request-ID", "attacker-controlled")
	handler.ServeHTTP(first, firstRequest)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	firstID := first.Header().Get("X-Request-ID")
	if firstID == "" || firstID == "attacker-controlled" {
		t.Fatalf("client request id was trusted: %q", firstID)
	}
	if firstID == second.Header().Get("X-Request-ID") {
		t.Fatalf("request id was reused: %q", firstID)
	}
}

func TestReadinessFailureDoesNotLeakDetails(t *testing.T) {
	handler := NewHandler(Config{
		Ready: func(context.Context) error { return errors.New("secret database detail") },
	})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(res.Body.String(), "secret") {
		t.Fatalf("readiness leaked internal error: %q", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"code":"service_unavailable"`) || !strings.Contains(res.Body.String(), `"request_id":"`) {
		t.Fatalf("readiness error is not machine-readable: %q", res.Body.String())
	}
}

func TestProtectedRouteAuthorization(t *testing.T) {
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("admin area"))
	})

	handler := NewHandler(Config{
		Routes: []router.Route{
			{Pattern: "GET /admin", Access: router.Protected, Handler: adminHandler},
		},
		Authorizer: func(r *http.Request) bool {
			return r.Header.Get("X-Test-Auth") == "authorized"
		},
	})

	// Unauthorized request
	reqUnauth := httptest.NewRequest(http.MethodGet, "/admin", nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recUnauth.Code)
	}
	if got := recUnauth.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("unauthorized content type = %q, want JSON", got)
	}
	if !strings.Contains(recUnauth.Body.String(), `"code":"unauthorized"`) {
		t.Fatalf("unauthorized body = %q", recUnauth.Body.String())
	}

	// Authorized request
	reqAuth := httptest.NewRequest(http.MethodGet, "/admin", nil)
	reqAuth.Header.Set("X-Test-Auth", "authorized")
	recAuth := httptest.NewRecorder()
	handler.ServeHTTP(recAuth, reqAuth)
	if recAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recAuth.Code)
	}
}

func TestGeneralRequestBodyLimitIsEnforced(t *testing.T) {
	handler := NewHandler(Config{
		HTTPBodyLimit: 4,
		Routes: []router.Route{{
			Pattern: "GET /body",
			Access:  router.Public,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusNoContent)
			}),
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/body", strings.NewReader("too large"))
	req.ContentLength = int64(len("too large"))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(res.Body.String(), `"code":"request_entity_too_large"`) {
		t.Fatalf("body = %q", res.Body.String())
	}
}

func TestStructuredHTTPLogContainsRequestIDWithoutSensitiveData(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := NewHandler(Config{Logger: logger})
	req := httptest.NewRequest(http.MethodGet, "/healthz?token=secret", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	line := logs.String()
	if !strings.Contains(line, `"msg":"http_request"`) || !strings.Contains(line, `"request_id":"`) {
		t.Fatalf("structured log missing request fields: %q", line)
	}
	if strings.Contains(line, "secret") || strings.Contains(line, "token=") {
		t.Fatalf("structured log leaked query data: %q", line)
	}
}

func TestMetricsAreProtectedAndBounded(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.SetDatabaseStatsProvider(func() observability.DatabaseStats {
		return observability.DatabaseStats{MaxOpenConnections: 10, OpenConnections: 2, WaitCount: 3}
	})
	metrics.SetCacheStatsProvider(func() observability.CacheStats {
		return observability.CacheStats{Hits: 4, Misses: 2, Evictions: 1}
	})
	metrics.SetJobStatsProvider(func() observability.JobStats {
		return observability.JobStats{Depth: 7, Leased: 2}
	})
	handler := NewHandler(Config{
		Metrics: metrics,
		Authorizer: func(r *http.Request) bool {
			return r.Header.Get("X-Test-Auth") == "authorized"
		},
	})
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("metrics unauthorized status = %d", unauthorized.Code)
	}
	authorizedRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	authorizedRequest.Header.Set("X-Test-Auth", "authorized")
	authorized := httptest.NewRecorder()
	handler.ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusOK {
		t.Fatalf("metrics authorized status = %d", authorized.Code)
	}
	if !strings.Contains(authorized.Body.String(), "hydro_http_requests_total") || !strings.Contains(authorized.Body.String(), `class="2xx"`) || !strings.Contains(authorized.Body.String(), "hydro_http_request_duration_seconds_bucket") || !strings.Contains(authorized.Body.String(), "hydro_database_pool_max_open_connections 10") || !strings.Contains(authorized.Body.String(), "hydro_cache_hits_total 4") || !strings.Contains(authorized.Body.String(), "hydro_jobs_depth 7") || !strings.Contains(authorized.Body.String(), "hydro_jobs_leased 2") {
		t.Fatalf("metrics output missing bounded counters: %q", authorized.Body.String())
	}
}

func TestMetricsIncludeMiddlewareRejectedRequests(t *testing.T) {
	metrics := observability.NewMetrics()
	handler := NewHandler(Config{
		HTTPBodyLimit: 4,
		Metrics:       metrics,
		Authorizer: func(*http.Request) bool {
			return true
		},
		Routes: []router.Route{{
			Pattern: "GET /limited",
			Access:  router.Public,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		}},
	})
	rejected := httptest.NewRequest(http.MethodGet, "/limited", strings.NewReader("too large"))
	rejected.ContentLength = int64(len("too large"))
	rejectedResponse := httptest.NewRecorder()
	handler.ServeHTTP(rejectedResponse, rejected)
	if rejectedResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("rejected status = %d", rejectedResponse.Code)
	}

	metricsRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResponse := httptest.NewRecorder()
	handler.ServeHTTP(metricsResponse, metricsRequest)
	if !strings.Contains(metricsResponse.Body.String(), `hydro_http_responses_total{class="4xx"} 1`) {
		t.Fatalf("metrics did not include middleware rejection: %q", metricsResponse.Body.String())
	}
}
