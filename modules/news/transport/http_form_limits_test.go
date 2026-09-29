package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func editorPost(path, body string) (*http.Request, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(tenant.WithScope(auth.WithUser(request.Context(), &auth.User{ID: "u1", Email: "editor@example.test", Role: auth.RoleEditor, Active: true}), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	return request, httptest.NewRecorder()
}

func TestAdminTaxonomyTermRejectsOversizedForm(t *testing.T) {
	h := Handler{}
	request, response := editorPost("/admin/news/taxonomy", "id=term-1&slug=term-1&kind=tag&labels="+strings.Repeat("x", 32<<10))
	h.adminTaxonomyTerm(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized taxonomy form status=%d body=%s", response.Code, response.Body.String())
	}
}
