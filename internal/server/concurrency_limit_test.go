package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConcurrencyLimitMiddlewareDisabledPassesThrough(t *testing.T) {
	called := false
	handler := ConcurrencyLimitMiddleware(0, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !called || rec.Code != http.StatusOK {
		t.Fatalf("expected pass-through with 200, got called=%v status=%d", called, rec.Code)
	}
}

func TestConcurrencyLimitMiddlewareShedsExcessRequests(t *testing.T) {
	release := make(chan struct{})
	var started sync.WaitGroup
	started.Add(1)
	handler := ConcurrencyLimitMiddleware(1, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started.Done()
		<-release
		w.WriteHeader(http.StatusOK)
	}))

	rec1 := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/", nil))
		close(done)
	}()
	started.Wait()

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec2.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 while at capacity, got %d", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on shed request")
	}

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not complete")
	}
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", rec1.Code)
	}
}

func TestConcurrencyLimitMiddlewareBypassesHealthChecksAtCapacity(t *testing.T) {
	release := make(chan struct{})
	var started sync.WaitGroup
	started.Add(1)
	bypass := func(r *http.Request) bool { return r.URL.Path == "/healthz" }
	handler := ConcurrencyLimitMiddleware(1, bypass)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		started.Done()
		<-release
		w.WriteHeader(http.StatusOK)
	}))

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		close(done)
	}()
	started.Wait()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("expected /healthz to bypass the limit while at capacity, got %d", health.Code)
	}

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not complete")
	}
}
