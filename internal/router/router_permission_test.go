package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectedRoutePermissionIsEnforced(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	held := map[string]bool{}
	handler, err := NewWithPermissions(
		[]Route{{Pattern: "POST /api/v1/admin/thing", Access: Protected, Permission: "admin.thing.write", Handler: ok}},
		func(*http.Request) bool { return true },
		func(_ *http.Request, permission string) bool { return held[permission] },
	)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/thing", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("signed in without the permission: %d, want 403", rec.Code)
	}
	held["admin.thing.write"] = true
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/thing", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("with the permission: %d, want 204", rec.Code)
	}
}

func TestADeclaredPermissionWithoutACheckerFailsAtStartup(t *testing.T) {
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if _, err := New([]Route{{Pattern: "GET /x", Access: Protected, Permission: "p", Handler: ok}}, func(*http.Request) bool { return true }); err == nil {
		t.Fatal("a permission that nothing can check must not be silently ignored")
	}
	if _, err := NewWithPermissions([]Route{{Pattern: "GET /y", Access: Public, Permission: "p", Handler: ok}}, nil, nil); err == nil {
		t.Fatal("a public route with a permission is a contradiction and must be refused")
	}
}
