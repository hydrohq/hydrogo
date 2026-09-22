package v1

import (
	"context"
	"testing"
)

func TestTenantScopeFailsClosedAndNamespacesKeys(t *testing.T) {
	if _, err := TenantScopeFromContext(context.Background()); err != ErrMissingTenantScope {
		t.Fatalf("missing scope error = %v", err)
	}
	scope, err := NewTenantScope("tenant-a", "site-a")
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithTenantScope(context.Background(), scope)
	key, err := TenantCacheKey(ctx, "news", "article-1")
	if err != nil || key != "t:tenant-a:news:article-1" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	if _, err := NewTenantScope("tenant/a", "site-a"); err == nil {
		t.Fatal("invalid tenant scope accepted")
	}
}
