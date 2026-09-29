package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/router"
)

func TestPublicErrorPagesReplacePublic404And500(t *testing.T) {
	render := func(_ context.Context, status int) ([]byte, bool) {
		return []byte("friendly-" + http.StatusText(status)), true
	}
	handler := NewHandler(Config{PublicErrorPage: render, Routes: []router.Route{
		{Pattern: "GET /missing", Access: router.Public, Handler: http.NotFoundHandler()},
		{Pattern: "GET /broken", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "database password=secret", http.StatusInternalServerError)
		})},
		{Pattern: "GET /panic", Access: router.Public, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(errors.New("stack details")) })},
		{Pattern: "GET /api/v1/broken", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "api error", http.StatusInternalServerError)
		})},
	}})
	for _, tc := range []struct {
		path   string
		status int
		want   string
	}{{"/missing", 404, "friendly-Not Found"}, {"/broken", 500, "friendly-Internal Server Error"}, {"/panic", 500, "friendly-Internal Server Error"}} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest("GET", tc.path, nil))
		if recorder.Code != tc.status || !strings.Contains(recorder.Body.String(), tc.want) || strings.Contains(recorder.Body.String(), "secret") || strings.Contains(recorder.Body.String(), "stack details") {
			t.Errorf("%s: status=%d body=%q", tc.path, recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(recorder.Header().Get("Content-Type"), "text/html") {
			t.Errorf("%s headers=%v", tc.path, recorder.Header())
		}
	}
	api := httptest.NewRecorder()
	handler.ServeHTTP(api, httptest.NewRequest("GET", "/api/v1/broken", nil))
	if api.Code != 500 || strings.Contains(api.Body.String(), "friendly-Internal Server Error") {
		t.Fatalf("API error was rewritten: status=%d body=%q", api.Code, api.Body.String())
	}
}
