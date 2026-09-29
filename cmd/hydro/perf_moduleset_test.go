package main

import (
	"regexp"
	"testing"
)

// Regression/benchmark for the per-request repeated-work fix in main.go:
// handlers used to call enabledModuleSet(cfg.Modules) / moduleEnabled(cfg.Modules, ...)
// on every request even though the module set is built once at startup
// (modules := enabledModuleSet(cfg.Modules)).

var configuredModulesFixtures = []string{"pages", " news ", "members", "analytics", "search", "commerce", "jobs", "media", "forms"}

func BenchmarkEnabledModuleSetPerRequest(b *testing.B) {
	modules := enabledModuleSet(configuredModulesFixtures)
	for i := 0; i < b.N; i++ {
		// The removed pattern: rebuild the module map on every "request".
		if !enabledModuleSet(configuredModulesFixtures)["pages"] {
			b.Fatal("pages should be enabled")
		}
	}
	_ = modules
}

func BenchmarkPrebuiltModuleMapLookup(b *testing.B) {
	modules := enabledModuleSet(configuredModulesFixtures)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !modules["pages"] {
			b.Fatal("pages should be enabled")
		}
	}
}

func TestFixturesCanonicalizeLikeMainPattern(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9]{12}_[a-z][a-z0-9_]{1,62}$`)
	if !pattern.MatchString("202601010000_pages") {
		t.Fatalf("migration pattern fixture failed")
	}
	if !moduleEnabled(configuredModulesFixtures, "PAGES") || moduleEnabled(configuredModulesFixtures, "forms-disabled") {
		t.Fatalf("moduleEnabled fixture inconsistent")
	}
}
