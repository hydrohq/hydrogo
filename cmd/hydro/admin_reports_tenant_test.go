package main

import (
	"net/http/httptest"
	"testing"

	"github.com/hydrohq/hydrogo/internal/tenant"
)

// The report tenant must come from the request scope: a tenant_id query
// parameter used to let one tenant's admin read another tenant's report.
func TestContentViewFilterIgnoresTenantQueryParameter(t *testing.T) {
	scope, err := tenant.NewScope("tenant-a", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/v1/admin/reports?tenant_id=tenant-b", nil)
	req = req.WithContext(tenant.WithScope(req.Context(), scope))
	if got := contentViewFilterFromRequest(req, 10).TenantID; got != "tenant-a" {
		t.Fatalf("scoped report tenant = %q, want tenant-a", got)
	}
	unscoped := httptest.NewRequest("GET", "/api/v1/admin/reports?tenant_id=tenant-b", nil)
	if got := contentViewFilterFromRequest(unscoped, 10).TenantID; got != "default" {
		t.Fatalf("single-tenant report tenant = %q, want default", got)
	}
}
