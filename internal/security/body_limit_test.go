package security

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestBodyLimitDefaultsToBoundedLimit(t *testing.T) {
	called := false
	handler := RequestBodyLimitMiddleware(0, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	body := strings.Repeat("x", int(DefaultRequestBodyLimit)+1)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
	if called {
		t.Fatal("handler was called for an oversized zero-config request")
	}
}

func TestRequestBodyLimitRejectsUnboundedIntegerBoundary(t *testing.T) {
	handler := RequestBodyLimitMiddleware(1<<63-1, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	body := strings.Repeat("x", int(DefaultRequestBodyLimit)+1)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRequestBodyLimitAllowsExplicitlyBoundedRequest(t *testing.T) {
	called := false
	handler := RequestBodyLimitMiddleware(4, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("ok"))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK || !called {
		t.Fatalf("status = %d, called = %v", res.Code, called)
	}
}
