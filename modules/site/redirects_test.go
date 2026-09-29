package site

import (
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestRedirectRegistryMatchesSafelyAndCountsByTenant(t *testing.T) {
	registry := &RedirectRegistry{RequireTenantScope: true}
	if err := registry.Replace("tenant-a", []RedirectRule{
		{ID: "wild", TenantID: "tenant-a", Pattern: "/old/*", Kind: "wildcard", TargetPath: "/new/*", Status: 301},
		{ID: "exact", TenantID: "tenant-a", Pattern: "/old/specific", Kind: "exact", TargetPath: "/preferred", Status: 302},
	}); err != nil {
		t.Fatal(err)
	}
	scopeA, _ := tenant.NewScope("tenant-a", "a")
	scopeB, _ := tenant.NewScope("tenant-b", "b")

	if _, ok := registry.Match(scopeB, "/old/specific"); ok {
		t.Fatal("redirect leaked across tenant scope")
	}
	got, ok := registry.Match(scopeA, "/old/specific")
	if !ok || got.TargetPath != "/preferred" || got.Status != 302 {
		t.Fatalf("exact rule should win: %#v, %v", got, ok)
	}
	got, ok = registry.Match(scopeA, "/old/section/page")
	if !ok || got.TargetPath != "/new/section/page" || got.Status != 301 {
		t.Fatalf("wildcard rule did not substitute the path: %#v, %v", got, ok)
	}
	for _, rule := range registry.List("tenant-a") {
		if rule.ID == "exact" && rule.Hits != 1 {
			t.Fatalf("exact hit count = %d, want 1", rule.Hits)
		}
		if rule.ID == "wild" && rule.Hits != 1 {
			t.Fatalf("wildcard hit count = %d, want 1", rule.Hits)
		}
	}
}

func TestRedirectRegistryRejectsInvalidRuleBeforeReplacingSnapshot(t *testing.T) {
	registry := &RedirectRegistry{}
	valid := RedirectRule{ID: "valid", Pattern: "/old", Kind: "exact", TargetPath: "/new", Status: 301}
	if err := registry.Replace("", []RedirectRule{valid}); err != nil {
		t.Fatal(err)
	}
	invalid := RedirectRule{ID: "bad", Pattern: "/bad", Kind: "exact", TargetPath: "https://example.test", Status: 301}
	if err := registry.Replace("", []RedirectRule{invalid}); err == nil {
		t.Fatal("external target was accepted")
	}
	if _, ok := registry.Match(tenant.Scope{}, "/old"); !ok {
		t.Fatal("invalid replacement removed the existing snapshot")
	}
}

func TestRedirectRegistryExpandsRegexCapture(t *testing.T) {
	registry := &RedirectRegistry{}
	rule := RedirectRule{ID: "regex", Pattern: `/legacy/(.*)`, Kind: "regex", TargetPath: "/articles/$1", Status: 301}
	if err := registry.Replace("", []RedirectRule{rule}); err != nil {
		t.Fatal(err)
	}
	got, ok := registry.Match(tenant.Scope{}, "/legacy/story")
	if !ok || got.TargetPath != "/articles/story" {
		t.Fatalf("regex redirect = %#v, %v", got, ok)
	}
}

func TestRedirectRegistryRejectsCircularExactAndWildcardChains(t *testing.T) {
	cases := map[string][]RedirectRule{
		"exact":    {{ID: "one", Pattern: "/one", Kind: "exact", TargetPath: "/two", Status: 301}, {ID: "two", Pattern: "/two", Kind: "exact", TargetPath: "/one", Status: 301}},
		"wildcard": {{ID: "one", Pattern: "/old/*", Kind: "wildcard", TargetPath: "/new/*", Status: 301}, {ID: "two", Pattern: "/new/*", Kind: "wildcard", TargetPath: "/old/*", Status: 301}},
	}
	for name, rules := range cases {
		t.Run(name, func(t *testing.T) {
			if err := (&RedirectRegistry{}).Replace("", rules); !errors.Is(err, ErrInvalidRedirect) {
				t.Fatalf("cycle error=%v", err)
			}
		})
	}
}
