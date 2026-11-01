package tenant

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openGovernanceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "gov.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
	CREATE TABLE tenants (
		tenant_id TEXT PRIMARY KEY,
		state TEXT NOT NULL,
		default_locale TEXT NOT NULL,
		timezone TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE tenant_governance (
		tenant_id TEXT PRIMARY KEY,
		display_name TEXT NOT NULL DEFAULT '',
		subdomain TEXT NOT NULL DEFAULT '',
		custom_domain TEXT NOT NULL DEFAULT '',
		blueprint TEXT NOT NULL DEFAULT 'boutique',
		plan TEXT NOT NULL DEFAULT 'starter',
		lifecycle_state TEXT NOT NULL DEFAULT 'active',
		trial_ends_at TEXT NOT NULL DEFAULT '',
		subscription_ends_at TEXT NOT NULL DEFAULT '',
		max_storage_bytes INTEGER NOT NULL DEFAULT 536870912,
		max_records INTEGER NOT NULL DEFAULT 5000,
		rate_limit_rps INTEGER NOT NULL DEFAULT 30,
		max_concurrent_reqs INTEGER NOT NULL DEFAULT 15,
		max_upload_bytes INTEGER NOT NULL DEFAULT 10485760,
		ai_enabled INTEGER NOT NULL DEFAULT 1,
		ai_provider TEXT NOT NULL DEFAULT 'gemini',
		ai_model TEXT NOT NULL DEFAULT 'gemini-2.5-flash',
		ai_encrypted_api_key TEXT NOT NULL DEFAULT '',
		ai_monthly_token_limit INTEGER NOT NULL DEFAULT 500000,
		ai_tokens_used INTEGER NOT NULL DEFAULT 0,
		ai_token_period TEXT NOT NULL DEFAULT '',
		owner_phone TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		dns_verified INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestGovernanceCRUDAndDomainResolution(t *testing.T) {
	ctx := context.Background()
	db := openGovernanceTestDB(t)

	gov := DefaultGovernance("ipek-butik")
	gov.DisplayName = "İpek Butik"
	gov.Subdomain = "ipekbutik"
	gov.CustomDomain = "www.ipekbutik.com"
	ApplyPlanPreset(&gov, PlanPro)

	if err := UpsertGovernance(ctx, db, "sqlite", gov); err != nil {
		t.Fatalf("UpsertGovernance: %v", err)
	}

	loaded, err := GetGovernance(ctx, db, "sqlite", "ipek-butik")
	if err != nil {
		t.Fatalf("GetGovernance: %v", err)
	}
	if loaded.DisplayName != "İpek Butik" || loaded.Plan != PlanPro || loaded.MaxRecords != 25000 {
		t.Fatalf("unexpected governance: %+v", loaded)
	}

	// Resolve by subdomain and custom domain
	scope, ok := ResolveTenantByDomain(ctx, db, "sqlite", "ipekbutik.platform.example.com")
	if !ok || scope.TenantID != "ipek-butik" {
		t.Fatalf("ResolveTenantByDomain subdomain failed: ok=%v scope=%+v", ok, scope)
	}
	scope, ok = ResolveTenantByDomain(ctx, db, "sqlite", "www.ipekbutik.com")
	if !ok || scope.TenantID != "ipek-butik" {
		t.Fatalf("ResolveTenantByDomain custom domain failed: ok=%v scope=%+v", ok, scope)
	}

	// Caddy On-Demand TLS check
	allowed, err := CheckDomainAllowed(ctx, db, "sqlite", "www.ipekbutik.com")
	if err != nil || !allowed {
		t.Fatalf("CheckDomainAllowed custom_domain: allowed=%v err=%v", allowed, err)
	}
	allowed, err = CheckDomainAllowed(ctx, db, "sqlite", "unknown.example.com")
	if err != nil || allowed {
		t.Fatalf("CheckDomainAllowed unknown should be false: allowed=%v err=%v", allowed, err)
	}
}

func TestGovernorLifecycleAndSubscriptionExpiry(t *testing.T) {
	ctx := context.Background()
	db := openGovernanceTestDB(t)

	// 1. Expired subscription automatically degrades active -> readonly
	gov := DefaultGovernance("expired-shop")
	gov.SubscriptionEndsAt = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	if err := UpsertGovernance(ctx, db, "sqlite", gov); err != nil {
		t.Fatal(err)
	}

	governor := NewGovernor(db, "sqlite")
	scope, _ := NewScope("expired-shop", "expired-shop")
	handler := governor.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Public GET must succeed even when expired (storefront stays online in readonly mode)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/content/products", nil)
	reqGet = reqGet.WithContext(WithScope(ctx, scope))
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected GET in readonly mode to return 200, got %d", recGet.Code)
	}

	// Admin/Content POST must be rejected with 402 Payment Required
	reqPost := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/products", strings.NewReader(`{}`))
	reqPost = reqPost.WithContext(WithScope(ctx, scope))
	recPost := httptest.NewRecorder()
	handler.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusPaymentRequired {
		t.Fatalf("expected POST in expired/readonly mode to return 402, got %d: %s", recPost.Code, recPost.Body.String())
	}

	// 2. Suspended workspace blocks all tenant traffic with 503, except orchestrator routes
	gov.LifecycleState = LifecycleSuspended
	if err := UpsertGovernance(ctx, db, "sqlite", gov); err != nil {
		t.Fatal(err)
	}
	governor.Invalidate("expired-shop")

	recSuspended := httptest.NewRecorder()
	handler.ServeHTTP(recSuspended, reqGet)
	if recSuspended.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected GET on suspended workspace to return 503, got %d", recSuspended.Code)
	}

	// Browser GET request with Accept: text/html must receive beautifully formatted HTML error page, not raw JSON
	reqBrowser := httptest.NewRequest(http.MethodGet, "/", nil)
	reqBrowser.Header.Set("Accept", "text/html,application/xhtml+xml")
	reqBrowser.Header.Set("X-Request-Id", "req-test-12345")
	reqBrowser = reqBrowser.WithContext(WithScope(ctx, scope))
	recBrowser := httptest.NewRecorder()
	handler.ServeHTTP(recBrowser, reqBrowser)
	if recBrowser.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected browser request on suspended workspace to return 503, got %d", recBrowser.Code)
	}
	if !strings.Contains(recBrowser.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected Content-Type text/html, got %s", recBrowser.Header().Get("Content-Type"))
	}
	if !strings.Contains(recBrowser.Body.String(), "Bu Mağaza Geçici Olarak Askıya Alınmıştır") {
		t.Fatalf("expected HTML body to contain explanation title, got: %s", recBrowser.Body.String())
	}
	if !strings.Contains(recBrowser.Body.String(), "req-test-12345") {
		t.Fatalf("expected HTML body to contain request ID, got: %s", recBrowser.Body.String())
	}

	reqOrch := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orchestrator/tenants/expired-shop", strings.NewReader(`{}`))
	reqOrch = reqOrch.WithContext(WithScope(ctx, scope))
	recOrch := httptest.NewRecorder()
	handler.ServeHTTP(recOrch, reqOrch)
	if recOrch.Code != http.StatusOK {
		t.Fatalf("expected orchestrator endpoint to bypass suspension, got %d", recOrch.Code)
	}
}

func TestGovernorConcurrencyAndAITokenMetering(t *testing.T) {
	ctx := context.Background()
	db := openGovernanceTestDB(t)

	gov := DefaultGovernance("busy-shop")
	gov.MaxConcurrentReqs = 1
	gov.AIMonthlyTokenLimit = 1000
	if err := UpsertGovernance(ctx, db, "sqlite", gov); err != nil {
		t.Fatal(err)
	}

	// Test AI token consumption and quota limit
	if _, err := ConsumeAITokens(ctx, db, "sqlite", "busy-shop", 600); err != nil {
		t.Fatalf("first ConsumeAITokens failed: %v", err)
	}
	if _, err := ConsumeAITokens(ctx, db, "sqlite", "busy-shop", 500); err == nil {
		t.Fatal("expected second ConsumeAITokens (600+500 > 1000) to fail with quota error")
	}

	// Demo workspaces are live trial stores: they consume AI quota like active
	// ones instead of failing every call, while suspended workspaces must not.
	demoGov := DefaultGovernance("demo-shop")
	demoGov.LifecycleState = LifecycleDemo
	demoGov.AIMonthlyTokenLimit = 1000
	if err := UpsertGovernance(ctx, db, "sqlite", demoGov); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeAITokens(ctx, db, "sqlite", "demo-shop", 100); err != nil {
		t.Fatalf("demo workspace AI consumption failed: %v", err)
	}
	suspendedGov := DefaultGovernance("suspended-shop")
	suspendedGov.LifecycleState = LifecycleSuspended
	if err := UpsertGovernance(ctx, db, "sqlite", suspendedGov); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeAITokens(ctx, db, "sqlite", "suspended-shop", 0); err == nil {
		t.Fatal("expected suspended workspace AI consumption to fail")
	}

	// Usage recorded under a previous calendar month must not carry over: the
	// counter rolls to zero on first consumption after the rollover.
	rolledGov := DefaultGovernance("rolled-shop")
	rolledGov.AIMonthlyTokenLimit = 1000
	rolledGov.AITokensUsed = 950
	rolledGov.AITokenPeriod = "2000-01"
	if err := UpsertGovernance(ctx, db, "sqlite", rolledGov); err != nil {
		t.Fatal(err)
	}
	afterReset, err := ConsumeAITokens(ctx, db, "sqlite", "rolled-shop", 50)
	if err != nil {
		t.Fatalf("consumption after monthly rollover failed: %v", err)
	}
	if afterReset.AITokensUsed != 50 {
		t.Fatalf("expected counter to reset then consume 50, got %d", afterReset.AITokensUsed)
	}
	if afterReset.AITokenPeriod == "2000-01" {
		t.Fatal("expected ai_token_period to advance to the current month")
	}

	// Test concurrency semaphore
	governor := NewGovernor(db, "sqlite")
	scope, _ := NewScope("busy-shop", "busy-shop")
	inHandler := make(chan struct{})
	releaseHandler := make(chan struct{})

	handler := governor.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case inHandler <- struct{}{}:
		default:
		}
		<-releaseHandler
		w.WriteHeader(http.StatusOK)
	}))

	done := make(chan int, 1)
	go func() {
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/content/products", nil).WithContext(WithScope(ctx, scope))
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req1)
		done <- rec1.Code
	}()

	<-inHandler

	// Second concurrent request while first is still in flight must return 429
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/content/products", nil).WithContext(WithScope(ctx, scope))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second concurrent request to return 429, got %d", rec2.Code)
	}

	close(releaseHandler)
	if code := <-done; code != http.StatusOK {
		t.Fatalf("expected first request to complete with 200, got %d", code)
	}
}
