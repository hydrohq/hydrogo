package main

import "testing"

func TestPushRoutesSkipWhenSafeModeLeavesNoDispatcher(t *testing.T) {
	app := &serveApp{modules: map[string]bool{"push": true}}
	if routes := app.pushRoutes(); routes != nil {
		t.Fatalf("routes = %v, want none without a dispatcher", routes)
	}
}

func TestAuditExplorerRouteExistsWithoutThePagesModule(t *testing.T) {
	app := &serveApp{modules: map[string]bool{"core": true, "auth": true, "collections": true}}
	for _, route := range app.adminRoutes() {
		if route.Pattern == "GET /api/v1/admin/audit" {
			return
		}
	}
	t.Fatal("GET /api/v1/admin/audit is not routed when pages is off; the Audit screen would 404 in crm and erp-lite")
}
