package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostResolverEstablishesScopeWithoutTrustingHeaders(t *testing.T) {
	scope, err := NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewHostResolver(map[string]Scope{"alpha.example.test": scope}, true)
	var got Scope
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = Require(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest("GET", "https://alpha.example.test/api/pages", nil)
	req.Host = "ALPHA.EXAMPLE.TEST:443"
	req.Header.Set("X-Tenant-ID", "tenant-attacker")
	recorder := httptest.NewRecorder()
	resolver.Middleware(next).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent || got.TenantID != "tenant-a" {
		t.Fatalf("status=%d scope=%#v", recorder.Code, got)
	}
}

func TestHostResolverRejectsPortSuffixGarbage(t *testing.T) {
	scope, _ := NewScope("tenant-a", "alpha")
	resolver := NewHostResolver(map[string]Scope{"alpha.example.test": scope}, true)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sc, _ := Require(r.Context())
		t.Fatalf("unexpected downstream request with scope %#v", sc)
	})
	req := httptest.NewRequest("GET", "https://alpha.example.test/api/pages", nil)
	// Attacker-controlled Host: the allowlisted hostname is grafted onto an
	// attacker-controlled suffix. Must not normalize back to the allowlisted
	// host and must not grant the tenant scope.
	req.Host = "alpha.example.test:443.evil.test"
	recorder := httptest.NewRecorder()
	resolver.Middleware(next).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusMisdirectedRequest {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusMisdirectedRequest)
	}
}

func TestHostResolverFailsClosedForUnknownRequiredHost(t *testing.T) {
	scope, _ := NewScope("tenant-a", "alpha")
	resolver := NewHostResolver(map[string]Scope{"alpha.example.test": scope}, true)
	recorder := httptest.NewRecorder()
	resolver.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected downstream request") })).ServeHTTP(recorder, httptest.NewRequest("GET", "https://unknown.example.test/", nil))
	if recorder.Code != http.StatusMisdirectedRequest {
		t.Fatalf("status=%d", recorder.Code)
	}
}

func TestHostResolverRuntimeSetRemoveAndFallbackCache(t *testing.T) {
	scopeA, _ := NewScope("tenant-a", "alpha")
	scopeB, _ := NewScope("tenant-b", "beta")
	resolver := NewHostResolver(map[string]Scope{"alpha.example.test": scopeA}, true)

	var fallbackCalls atomic.Int32
	resolver.SetFallback(func(ctx context.Context, host string) (Scope, bool) {
		fallbackCalls.Add(1)
		if host == "dynamic.example.test" {
			return scopeB, true
		}
		return Scope{}, false
	}, time.Minute)

	// 1. Dynamic fallback resolves and caches.
	for i := 0; i < 3; i++ {
		got, ok := resolver.Resolve(context.Background(), "dynamic.example.test:443")
		if !ok || got.TenantID != "tenant-b" {
			t.Fatalf("resolve dynamic: ok=%v scope=%#v", ok, got)
		}
	}
	if fallbackCalls.Load() != 1 {
		t.Fatalf("fallback called %d times, want 1 (cached)", fallbackCalls.Load())
	}

	// 2. Runtime SetHost registers a brand-new host without restart.
	scopeC, _ := NewScope("tenant-c", "gamma")
	if !resolver.SetHost("gamma.example.test", scopeC) {
		t.Fatal("SetHost returned false")
	}
	if !resolver.Matches("GAMMA.EXAMPLE.TEST:8080") {
		t.Fatal("expected Matches to succeed for runtime host")
	}
	resolver.RemoveHost("gamma.example.test")
	if resolver.Matches("gamma.example.test") {
		t.Fatal("expected Matches to return false after RemoveHost")
	}
}
