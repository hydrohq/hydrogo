package contract_test

import (
	"context"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func FuzzTenantScopeCacheIsolation(f *testing.F) {
	f.Add("alpha", "beta", "pages", "home")
	f.Fuzz(func(t *testing.T, first, second, module, key string) {
		firstScope, err := tenant.NewScope(first, first)
		if err != nil {
			return
		}
		secondScope, err := tenant.NewScope(second, second)
		if err != nil {
			return
		}
		firstKey, err := tenant.CacheKey(tenant.WithScope(context.Background(), firstScope), module, key)
		if err != nil {
			return
		}
		secondKey, err := tenant.CacheKey(tenant.WithScope(context.Background(), secondScope), module, key)
		if err != nil {
			return
		}
		if first != second && firstKey == secondKey {
			t.Fatalf("tenant cache collision for %q and %q", first, second)
		}
		if _, err := tenant.CacheKey(context.Background(), module, key); err == nil {
			t.Fatal("missing scope accepted")
		}
	})
}
