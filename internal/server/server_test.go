package server

import (
	"context"
	"errors"
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

	// Authorized request
	reqAuth := httptest.NewRequest(http.MethodGet, "/admin", nil)
	reqAuth.Header.Set("X-Test-Auth", "authorized")
	recAuth := httptest.NewRecorder()
	handler.ServeHTTP(recAuth, reqAuth)
	if recAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recAuth.Code)
	}
}

func TestMetricsAreProtectedAndBounded(t *testing.T) {
	metrics := observability.NewMetrics()
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
	if !strings.Contains(authorized.Body.String(), "hydro_http_requests_total") || !strings.Contains(authorized.Body.String(), `class="2xx"`) {
		t.Fatalf("metrics output missing bounded counters: %q", authorized.Body.String())
	}
}
