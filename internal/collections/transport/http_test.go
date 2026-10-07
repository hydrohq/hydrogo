package transport

import (
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

func TestCollectionRoutesRequireAdmin(t *testing.T) {
	handler := Handler{Service: collections.Service{}}
	for _, route := range handler.Routes() {
		if route.Access != router.Protected {
			t.Errorf("route %q access = %v, want protected", route.Pattern, route.Access)
		}
		for _, role := range []auth.Role{auth.RoleEditor, auth.RoleViewer} {
			t.Run(route.Pattern+"/"+string(role), func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "staff", Role: role, Active: true}))
				recorder := httptest.NewRecorder()
				route.Handler.ServeHTTP(recorder, req)
				if recorder.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want forbidden", recorder.Code)
				}
			})
		}
	}
}

func TestSchemaBoundParsing(t *testing.T) {
	if value, err := parseOptionalBound(""); err != nil || value != nil {
		t.Fatalf("empty bound = %v, %v", value, err)
	}
	value, err := parseOptionalBound("-2.5")
	if err != nil || value == nil || *value != -2.5 {
		t.Fatalf("bound = %v, %v", value, err)
	}
	if _, err := parseOptionalBound("not-a-number"); err == nil {
		t.Fatal("invalid bound accepted")
	}
	if _, err := parseOptionalBound("NaN"); err != nil {
		t.Fatalf("NaN is rejected by schema validation, parser should preserve it: %v", err)
	}
	if value, _ := parseOptionalBound("NaN"); !math.IsNaN(*value) {
		t.Fatalf("parsed bound = %v", *value)
	}
}

func TestDeleteSchemaRequiresExplicitConfirmation(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/collections/articles/delete", strings.NewReader("confirm_schema=other"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("collection", "articles")
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	recorder := httptest.NewRecorder()
	(Handler{}).adminDeleteSchema(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want bad request", recorder.Code)
	}
}

func TestAPIErrorDoesNotExposeInternalDetails(t *testing.T) {
	recorder := httptest.NewRecorder()
	apiError(recorder, errors.New("postgres connection string leaked"))
	if strings.Contains(recorder.Body.String(), "postgres connection") || !strings.Contains(recorder.Body.String(), "collection request could not be completed") {
		t.Fatalf("unsafe error response: %s", recorder.Body.String())
	}
}
