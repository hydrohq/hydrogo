package tenant

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestScopeRequiresExplicitValidatedContext(t *testing.T) {
	if _, err := Require(context.Background()); !errors.Is(err, ErrMissingScope) {
		t.Fatalf("missing scope error = %v", err)
	}
	scope, err := NewScope("tenant-aaa", "client-alpha")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Require(WithScope(context.Background(), scope))
	if err != nil || got != scope {
		t.Fatalf("scope = %#v, err = %v", got, err)
	}
}

func TestScopeRejectsKeyInjectionAndOversizedComponents(t *testing.T) {
	for _, tc := range []struct {
		name, id, slug string
	}{
		{name: "separator", id: "tenant:evil", slug: "valid"},
		{name: "whitespace", id: "tenant", slug: "bad slug"},
		{name: "unicode", id: "tenant-ä", slug: "valid"},
		{name: "empty", id: "", slug: "valid"},
		{name: "oversized", id: strings.Repeat("a", maxComponentLength+1), slug: "valid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewScope(tc.id, tc.slug); !errors.Is(err, ErrInvalidScope) {
				t.Fatalf("error = %v, want ErrInvalidScope", err)
			}
		})
	}
}

func TestCacheKeyIsTenantScopedAndFailClosed(t *testing.T) {
	a, _ := NewScope("tenant-a", "alpha")
	b, _ := NewScope("tenant-b", "beta")
	keyA, err := CacheKey(WithScope(context.Background(), a), "pages", "home")
	if err != nil {
		t.Fatal(err)
	}
	keyB, err := CacheKey(WithScope(context.Background(), b), "pages", "home")
	if err != nil {
		t.Fatal(err)
	}
	if keyA == keyB || keyA != "t:tenant-a:pages:home" {
		t.Fatalf("cache keys = %q and %q", keyA, keyB)
	}
	if _, err := CacheKey(context.Background(), "pages", "home"); !errors.Is(err, ErrMissingScope) {
		t.Fatalf("missing scope cache key error = %v", err)
	}
}
