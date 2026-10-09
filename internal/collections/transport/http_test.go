package transport

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
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

func TestCollectionMutationRoutesEnforceCSRFAndAdminRole(t *testing.T) {
	routes := (Handler{Service: collections.Service{}}).Routes()
	protected, err := router.New(routes, func(r *http.Request) bool {
		user := auth.UserFromContext(r.Context())
		return user != nil && user.Active
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := security.CSRFMiddleware(false, nil)(protected)
	mutationCount := 0
	for _, route := range routes {
		method, path, ok := strings.Cut(route.Pattern, " ")
		if !ok || (method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && method != http.MethodDelete) {
			continue
		}
		mutationCount++
		path = strings.NewReplacer("{collection}", "articles", "{id}", "article-1").Replace(path)
		body := ""
		contentType := "application/json"
		if strings.HasPrefix(path, "/admin/") {
			contentType = "application/x-www-form-urlencoded"
			body = url.Values{"confirm_schema": {"articles"}, "confirm": {"yes"}}.Encode()
		}
		for _, tc := range []struct {
			name string
			user *auth.User
		}{
			{name: "csrf rejected", user: &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}},
			{name: "editor denied after csrf", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}},
			{name: "viewer denied after csrf", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}},
		} {
			t.Run(route.Pattern+"/"+tc.name, func(t *testing.T) {
				request := httptest.NewRequest(method, path, strings.NewReader(body))
				request.Header.Set("Content-Type", contentType)
				request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
				if tc.name != "csrf rejected" {
					request.Header.Set(security.CSRFHeaderName, "valid-token")
				} else {
					request.Header.Set(security.CSRFHeaderName, "wrong-token")
				}
				request = request.WithContext(auth.WithUser(context.Background(), tc.user))
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusForbidden {
					t.Fatalf("status=%d body=%q, want 403", response.Code, response.Body.String())
				}
			})
		}
	}
	if mutationCount != 9 {
		t.Fatalf("covered %d collection mutations, want all 9 registered POST/PUT/DELETE routes", mutationCount)
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
