package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

func TestCollectionRoutesRequireAdmin(t *testing.T) {
	handler := Handler{Service: collections.Service{}}
	for name, call := range map[string]http.HandlerFunc{
		"admin studio": handler.adminIndex,
		"schema api":   handler.apiSchemas,
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "staff", Role: auth.RoleViewer, Active: true}))
			recorder := httptest.NewRecorder()
			call.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want forbidden", recorder.Code)
			}
		})
	}
}
