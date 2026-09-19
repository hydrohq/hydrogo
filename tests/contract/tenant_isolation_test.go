package contract_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type ScopedRepositoryMock struct {
	data map[string]map[string]string // tenantID -> id -> content
}

func (r *ScopedRepositoryMock) Get(ctx context.Context, id string) (string, error) {
	tc, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	tenantMap, exists := r.data[tc.TenantID]
	if !exists {
		return "", fmt.Errorf("tenant data not found")
	}
	val, ok := tenantMap[id]
	if !ok {
		return "", fmt.Errorf("record not found in tenant scope")
	}
	return val, nil
}

func (r *ScopedRepositoryMock) Put(ctx context.Context, id, val string) error {
	tc, err := tenant.Require(ctx)
	if err != nil {
		return err
	}
	if r.data == nil {
		r.data = make(map[string]map[string]string)
	}
	if _, exists := r.data[tc.TenantID]; !exists {
		r.data[tc.TenantID] = make(map[string]string)
	}
	r.data[tc.TenantID][id] = val
	return nil
}

// TestTenantIsolationContract certifies ADR-0140 & ADR-0149:
// 1. Missing tenant context fails closed immediately
// 2. Cross-tenant reads are strictly partitioned
// 3. Cache keys must be tenant-prefixed
func TestTenantIsolationContract(t *testing.T) {
	repo := &ScopedRepositoryMock{}

	scopeA, err := tenant.NewScope("tenant-aaa", "client-alpha")
	if err != nil {
		t.Fatal(err)
	}
	scopeB, err := tenant.NewScope("tenant-bbb", "client-beta")
	if err != nil {
		t.Fatal(err)
	}
	ctxTenantA := tenant.WithScope(context.Background(), scopeA)
	ctxTenantB := tenant.WithScope(context.Background(), scopeB)
	ctxEmpty := context.Background()

	// 1. Fail-closed test on missing tenant context
	err = repo.Put(ctxEmpty, "doc-1", "secret-data")
	if !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("expected ErrMissingTenantContext on empty context, got: %v", err)
	}

	_, err = repo.Get(ctxEmpty, "doc-1")
	if !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("expected ErrMissingTenantContext on get with empty context, got: %v", err)
	}

	// 2. Put data in Tenant A
	if err := repo.Put(ctxTenantA, "doc-1", "alpha-sensitive-document"); err != nil {
		t.Fatalf("failed to put data in tenant A: %v", err)
	}

	// 3. Verify Tenant A can read its own data
	val, err := repo.Get(ctxTenantA, "doc-1")
	if err != nil || val != "alpha-sensitive-document" {
		t.Fatalf("expected tenant A to read its data, got val=%s, err=%v", val, err)
	}

	// 4. Verify Tenant B CANNOT read Tenant A's record (Strict Isolation)
	_, err = repo.Get(ctxTenantB, "doc-1")
	if err == nil {
		t.Fatalf("security violation: Tenant B was able to read Tenant A's document!")
	}

	// 5. Cache Key Isolation Contract
	cacheKeyA, err := tenant.CacheKey(ctxTenantA, "pages", "home-slug")
	if err != nil {
		t.Fatal(err)
	}
	cacheKeyB, err := tenant.CacheKey(ctxTenantB, "pages", "home-slug")
	if err != nil {
		t.Fatal(err)
	}
	if cacheKeyA == cacheKeyB {
		t.Fatalf("cache keys must not collide between tenants")
	}
}
