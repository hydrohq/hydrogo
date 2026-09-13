package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
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
		if got := res.Header().Get("Permissions-Policy"); got != "camera=(), microphone=(), geolocation=()" {
			t.Errorf("GET %s permissions policy = %q", route.path, got)
		}
		if route.path != "/unregistered" && (res.Header().Get("Cache-Control") != "no-store" || res.Header().Get("Pragma") != "no-cache") {
			t.Errorf("GET %s is cacheable: cache-control=%q pragma=%q", route.path, res.Header().Get("Cache-Control"), res.Header().Get("Pragma"))
		}
		requestID := res.Header().Get("X-Request-ID")
		if len(requestID) != 32 {
			t.Errorf("GET %s request id length = %d, want 32", route.path, len(requestID))
		}
	}
}

func TestServeContextNormalization(t *testing.T) {
	if nonNilContext(nil) == nil {
		t.Fatal("nil context was not normalized")
	}
	ctx := context.Background()
	if nonNilContext(ctx) != ctx {
		t.Fatal("non-nil context was replaced")
	}
}

func TestServeGracefulShutdownWaitsForActiveHandler(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	release := make(chan struct{})
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- Serve(ctx, Config{Address: address, ShutdownTimeout: time.Second, Routes: []router.Route{{
			Pattern: "GET /slow", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				close(started)
				<-release
				w.WriteHeader(http.StatusNoContent)
			}),
		}}})
	}()
	client := &http.Client{Timeout: time.Second}
	for i := 0; i < 50; i++ {
		resp, requestErr := client.Get("http://" + address + "/healthz")
		if requestErr == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	requestDone := make(chan error, 1)
	go func() {
		resp, requestErr := client.Get("http://" + address + "/slow")
		if resp != nil {
			_ = resp.Body.Close()
		}
		requestDone <- requestErr
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("slow handler did not start")
	}
	cancel()
	select {
	case err := <-serveDone:
		t.Fatalf("server stopped before active handler drained: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("graceful shutdown error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after active handler drained")
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
	metrics := observability.NewMetrics()
	handler := NewHandler(Config{
		Metrics: metrics,
		Ready:   func(context.Context) error { return errors.New("secret database detail") },
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
	metricsResponse := httptest.NewRecorder()
	metrics.Handler(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metricsResponse.Body.String(), "hydro_readiness_failures_total 1\n") {
		t.Fatalf("readiness failure metric missing: %q", metricsResponse.Body.String())
	}
	if !strings.Contains(metricsResponse.Body.String(), "hydro_readiness_state 0\n") {
		t.Fatalf("readiness state metric missing: %q", metricsResponse.Body.String())
	}
}

func TestReadinessStateMetricRecoversAfterSuccessfulProbe(t *testing.T) {
	metrics := observability.NewMetrics()
	ready := false
	handler := NewHandler(Config{
		Metrics: metrics,
		Ready: func(context.Context) error {
			if !ready {
				return errors.New("not ready")
			}
			return nil
		},
	})

	failed := httptest.NewRecorder()
	handler.ServeHTTP(failed, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if failed.Code != http.StatusServiceUnavailable {
		t.Fatalf("initial readiness status = %d", failed.Code)
	}
	ready = true
	recovered := httptest.NewRecorder()
	handler.ServeHTTP(recovered, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if recovered.Code != http.StatusOK {
		t.Fatalf("recovered readiness status = %d", recovered.Code)
	}

	metricsResponse := httptest.NewRecorder()
	metrics.Handler(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsResponse.Body.String()
	if !strings.Contains(body, "hydro_readiness_failures_total 1\n") || !strings.Contains(body, "hydro_readiness_state 1\n") {
		t.Fatalf("readiness recovery metrics = %q", body)
	}
}

func TestUnexpectedHandlerPanicBecomesStructuredInternalError(t *testing.T) {
	handler := NewHandler(Config{Routes: []router.Route{{
		Pattern: "GET /panic",
		Access:  router.Public,
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("private panic detail")
		}),
	}}})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusInternalServerError)
	}
	if got := res.Header().Get("X-Request-ID"); len(got) != 32 {
		t.Fatalf("request id = %q, want 32 hex characters", got)
	}
	body := res.Body.String()
	if strings.Contains(body, "private panic detail") || !strings.Contains(body, `"code":"internal_server_error"`) || !strings.Contains(body, `"request_id":"`) {
		t.Fatalf("panic response leaked details or was not structured: %q", body)
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

func TestMiddlewareChainPreservesResponseFlushing(t *testing.T) {
	metrics := observability.NewMetrics()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := NewHandler(Config{
		Metrics: metrics,
		Logger:  logger,
		Routes: []router.Route{{
			Pattern: "GET /stream",
			Access:  router.Public,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				flusher, ok := w.(http.Flusher)
				if !ok {
					http.Error(w, "flush unavailable", http.StatusInternalServerError)
					return
				}
				flusher.Flush()
				_, _ = w.Write([]byte("ok"))
			}),
		}},
	})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if res.Code != http.StatusOK || res.Body.String() != "ok" {
		t.Fatalf("stream response = %d/%q", res.Code, res.Body.String())
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
	if got := authorized.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("metrics cache-control = %q", got)
	}
	if got := authorized.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("metrics pragma = %q", got)
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

func TestErrorMiddlewarePreservesRetryAfter(t *testing.T) {
	handler := NewHandler(Config{Routes: []router.Route{{
		Pattern: "GET /limited",
		Access:  router.Public,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			security.WriteRateLimitedResponse(w, "too many", time.Minute)
		}),
	}}})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/limited", nil))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", response.Code)
	}
	if got := response.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want 60", got)
	}
}
