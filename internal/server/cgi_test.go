package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCGIRequestShapeCompatibleWithHTTPHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://example.test/healthz", nil)
	response := httptest.NewRecorder()
	NewHandler(Config{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "ok") {
		t.Fatalf("GET /healthz = %d %q", response.Code, response.Body.String())
	}
}

func TestLazyCronRunsOnRequestAndContinues(t *testing.T) {
	runs := 0
	handler := LazyCron(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}), func(ctx context.Context) {
		if ctx == nil {
			t.Fatal("cron received nil request context")
		}
		runs++
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Body.String() != "ok" || runs != 1 {
		t.Fatalf("response=%q cron runs=%d", response.Body.String(), runs)
	}
}
