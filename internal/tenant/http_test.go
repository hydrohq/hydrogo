package tenant

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
