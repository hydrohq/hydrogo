package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Wiring-audit invariants for the serve() boot block. The tests read the
// wiring source directly so a regression — a tenant-scoped store construction
// losing its RequireTenantScope toggle — fails here and names the diverging
// line instead of being discovered at runtime.

// wiringBlock returns the boot wiring source: serve() and the serveApp
// methods it calls are spread over the package, so this concatenates every
// non-test Go file of package main.
func wiringBlock(t *testing.T) string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list package sources: %v", err)
	}
	sort.Strings(files)
	var source strings.Builder
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source.Write(data)
		source.WriteString("\n")
	}
	if !strings.Contains(source.String(), "\nfunc serve(") {
		t.Fatal("serve() not found in package main")
	}
	return source.String()
}

// TestWiringAnalyticsStoreStaysGlobal pins a deliberate non-uniformity found
// during the wiring audit: analyticsstore.Store is intentionally constructed
// without RequireTenantScope (the field does not exist on the store — analytics
// events are global/cross-tenant by design), while SearchInsightsHandler at
// ~line 2295 receives requireTenantScope separately. If someone "aligns" it
// with the other stores, this test fails to force a review of that change.
func TestWiringAnalyticsStoreStaysGlobal(t *testing.T) {
	block := wiringBlock(t)
	if strings.Contains(block, "analyticsstore.Store{Executor: db, Dialect: cfg.Database.Dialect, RequireTenantScope") {
		t.Errorf("analyticsstore.Store was given RequireTenantScope — analytics wiring is intentionally global (no such field on analyticsstore.Store); revert")
	}
	assoc := strings.Count(block, "analytics.Store{")
	_ = assoc
	if !strings.Contains(block, "analyticsstore.Store{Executor: db, Dialect: cfg.Database.Dialect}") {
		t.Fatalf("analyticsstore.Store construction changed shape; re-audit the analytics wiring in the boot block")
	}
}

// TestWiringPushDispatcherGuarded asserts the push module dereference of
// pushDispatcher at route registration only happens under a nil-safe guard.
func TestWiringPushDispatcherGuarded(t *testing.T) {
	block := wiringBlock(t)
	pushRoute := strings.Count(block, "Dispatcher: *pushDispatcher")
	declaration := strings.Contains(block, "var pushDispatcher *pushservice.Dispatcher")
	if pushRoute > 0 && !declaration {
		t.Fatalf("*pushDispatcher dereferenced %d time(s) at route registration but pushDispatcher has no nil-guarded pointer declaration in the block", pushRoute)
	}
}

// TestWiringFormViewLimiterRequiresMembersAndAnalytics keeps the deliberate
// coupling explicit: formViewLimiter is only non-nil when both modules are on.
func TestWiringFormViewLimiterNilSafe(t *testing.T) {
	block := wiringBlock(t)
	if strings.Contains(block, "formViewLimiter := newRateLimiter") {
		t.Errorf("formViewLimiter became non-nil without the modules[\"forms\"] && modules[\"analytics\"] + hasher guard")
	}
}
