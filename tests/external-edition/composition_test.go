package externaledition

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	generated "example.com/external-edition/generated"
	generatedarticle "example.com/external-edition/generated/article"
	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
	_ "modernc.org/sqlite"
)

func TestExternalEditionCanComposeGeneratedAdmin(t *testing.T) {
	manifest := []byte(`{"version":"v1","resources":[{"id":"projects","label":"Projects","fields":[{"id":"title","label":"Title","kind":"text","required":true,"visible_in_list":true}],"actions":[{"label":"Open","href":"/admin/projects"}]},{"id":"disabled","label":"Disabled","fields":[{"id":"title","label":"Title","kind":"text"}],"actions":[{"label":"Open","href":"/admin/disabled"}]}]}`)
	mux := http.NewServeMux()
	options := adminstudio.GeneratedAdminOptions{
		API: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/projects" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		}),
		Authorize:       func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		ResourceAccess: func(_ *http.Request, _ adminstudio.Collection, id string) adminstudio.ResourceAccess {
			return adminstudio.ResourceAccess{Create: id == "", Update: id != "", Delete: id != ""}
		},
		CSRFToken:    func(*http.Request) string { return "edition-token" },
		ValidateCSRF: func(*http.Request) bool { return true },
		RenderPage: adminstudio.StudioPageRenderer(func(*http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "edition-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}
	if err := adminstudio.RegisterGenerated(mux, manifest, []string{"projects"}, options); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/resources/projects", nil))
	for _, want := range []string{`<aside class="studio-sidebar">`, `data-studio-role="admin"`, `href="/admin/resources/projects"`, "Projects"} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Fatalf("shared generated Admin shell missing %q", want)
		}
	}
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Yeni kayıt") {
		t.Fatalf("registered generated Admin route did not render: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	inactive := httptest.NewRecorder()
	mux.ServeHTTP(inactive, httptest.NewRequest(http.MethodGet, "/admin/resources/disabled", nil))
	if inactive.Code != http.StatusNotFound {
		t.Fatalf("unselected generated resource route was registered: status=%d body=%s", inactive.Code, inactive.Body.String())
	}
}

type externalResourcePolicy struct{}

func (externalResourcePolicy) CanRead(context.Context, string) bool   { return true }
func (externalResourcePolicy) CanCreate(context.Context) bool         { return true }
func (externalResourcePolicy) CanUpdate(context.Context, string) bool { return true }
func (externalResourcePolicy) CanDelete(context.Context, string) bool { return true }

func TestExternalEditionGeneratedTenantCRUDAndAdminUseAggregateHelpers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), generatedarticle.MigrationSQL); err != nil {
		t.Fatalf("apply generated SQLite migration: %v", err)
	}

	missingTenantResolverMux := http.NewServeMux()
	if err := generated.RegisterGeneratedResources(missingTenantResolverMux, generated.GeneratedResourceOptions{
		DB: db, Dialect: "sqlite",
		PolicyFor: func(string) generated.GeneratedResourcePolicy { return externalResourcePolicy{} },
	}); err == nil {
		t.Fatal("tenant-scoped route registration accepted a missing tenant resolver")
	}
	missingResolverResponse := httptest.NewRecorder()
	missingTenantResolverMux.ServeHTTP(missingResolverResponse, httptest.NewRequest(http.MethodGet, "/api/v1/article", nil))
	if missingResolverResponse.Code != http.StatusNotFound {
		t.Fatalf("tenant route was partially registered without a resolver: status=%d", missingResolverResponse.Code)
	}

	mux := http.NewServeMux()
	if err := generated.RegisterGeneratedResources(mux, generated.GeneratedResourceOptions{
		DB: db, Dialect: "sqlite",
		PolicyFor: func(string) generated.GeneratedResourcePolicy { return externalResourcePolicy{} },
		ResolveTenant: func(resourceID string, r *http.Request) (string, bool) {
			if resourceID != "article" {
				return "", false
			}
			tenant := r.Header.Get("X-Test-Tenant")
			return tenant, tenant != ""
		},
	}); err != nil {
		t.Fatalf("register generated routes: %v", err)
	}
	if err := generated.RegisterGeneratedAdmin(mux, adminstudio.GeneratedAdminOptions{
		API: mux, Authorize: func(r *http.Request) bool { return r.Header.Get("X-Test-Role") != "viewer" },
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken:       func(*http.Request) string { return "edition-token" },
		ValidateCSRF:    func(r *http.Request) bool { return r.FormValue("_csrf") == "edition-token" },
		RenderPage: adminstudio.StudioPageRenderer(func(*http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "edition-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}); err != nil {
		t.Fatalf("register generated Admin: %v", err)
	}

	create := httptest.NewRequest(http.MethodPost, "/api/v1/article", strings.NewReader(`{"id":"article-1","title":"External tenant record"}`))
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("X-Test-Tenant", "tenant-a")
	created := httptest.NewRecorder()
	mux.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("generated tenant create status=%d body=%s", created.Code, created.Body.String())
	}

	readResourceFor := func(tenant, id string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/article/"+id, nil)
		req.Header.Set("X-Test-Tenant", tenant)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	readFor := func(tenant string) *httptest.ResponseRecorder { return readResourceFor(tenant, "article-1") }
	if response := readFor("tenant-a"); response.Code != http.StatusOK {
		t.Fatalf("same-tenant read status=%d body=%s", response.Code, response.Body.String())
	}
	if response := readFor("tenant-b"); response.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant read status=%d body=%s", response.Code, response.Body.String())
	}

	for _, record := range []struct{ tenant, id, title string }{
		{"tenant-a", "tenant-a-row", "Tenant A record"},
		{"tenant-b", "tenant-b-row", "Tenant B record"},
	} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/article", strings.NewReader(`{"id":"`+record.id+`","title":"`+record.title+`"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Test-Tenant", record.tenant)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s API record status=%d body=%s", record.tenant, response.Code, response.Body.String())
		}
	}
	adminList := func(tenant string) string {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil)
		request.Header.Set("X-Test-Tenant", tenant)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s Admin list status=%d body=%s", tenant, response.Code, response.Body.String())
		}
		return response.Body.String()
	}
	if body := adminList("tenant-a"); !strings.Contains(body, "Tenant A record") || strings.Contains(body, "Tenant B record") {
		t.Fatalf("tenant A Admin list leaked or omitted a record: %s", body)
	}
	if body := adminList("tenant-b"); !strings.Contains(body, "Tenant B record") || strings.Contains(body, "Tenant A record") {
		t.Fatalf("tenant B Admin list leaked or omitted a record: %s", body)
	}
	viewerList := httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil)
	viewerList.Header.Set("X-Test-Role", "viewer")
	viewerListResponse := httptest.NewRecorder()
	mux.ServeHTTP(viewerListResponse, viewerList)
	if viewerListResponse.Code != http.StatusForbidden {
		t.Fatalf("generated Admin allowed a viewer role: status=%d body=%s", viewerListResponse.Code, viewerListResponse.Body.String())
	}
	viewerMutation := httptest.NewRequest(http.MethodPost, "/admin/resources/article/new", strings.NewReader("_csrf=edition-token&id=viewer-created&title=Must+be+denied"))
	viewerMutation.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	viewerMutation.Header.Set("X-Test-Role", "viewer")
	viewerMutation.Header.Set("X-Test-Tenant", "tenant-a")
	viewerMutationResponse := httptest.NewRecorder()
	mux.ServeHTTP(viewerMutationResponse, viewerMutation)
	if viewerMutationResponse.Code != http.StatusForbidden {
		t.Fatalf("generated Admin allowed a viewer mutation: status=%d body=%s", viewerMutationResponse.Code, viewerMutationResponse.Body.String())
	}
	if response := readResourceFor("tenant-a", "viewer-created"); response.Code != http.StatusNotFound {
		t.Fatalf("viewer mutation was persisted: status=%d body=%s", response.Code, response.Body.String())
	}

	adminMutation := func(path string, values url.Values) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("X-Test-Tenant", "tenant-a")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	newForm := httptest.NewRequest(http.MethodGet, "/admin/resources/article/new", nil)
	newForm.Header.Set("X-Test-Tenant", "tenant-a")
	newFormResponse := httptest.NewRecorder()
	mux.ServeHTTP(newFormResponse, newForm)
	if newFormResponse.Code != http.StatusOK || !strings.Contains(newFormResponse.Body.String(), `name="_csrf" value="edition-token"`) {
		t.Fatalf("generated Admin create form did not render a CSRF token: status=%d body=%s", newFormResponse.Code, newFormResponse.Body.String())
	}
	createdByAdmin := adminMutation("/admin/resources/article/new", url.Values{"_csrf": {"edition-token"}, "id": {"admin-created"}, "title": {"Created in Admin"}})
	if createdByAdmin.Code != http.StatusSeeOther {
		t.Fatalf("generated Admin create status=%d body=%s", createdByAdmin.Code, createdByAdmin.Body.String())
	}
	updatedByAdmin := adminMutation("/admin/resources/article/admin-created", url.Values{"_csrf": {"edition-token"}, "title": {"Updated in Admin"}})
	if updatedByAdmin.Code != http.StatusSeeOther {
		t.Fatalf("generated Admin update status=%d body=%s", updatedByAdmin.Code, updatedByAdmin.Body.String())
	}
	if body := adminList("tenant-a"); !strings.Contains(body, "Updated in Admin") {
		t.Fatalf("Admin update was not reflected in its tenant list: %s", body)
	}
	deletedByAdmin := adminMutation("/admin/resources/article/admin-created/delete", url.Values{"_csrf": {"edition-token"}})
	if deletedByAdmin.Code != http.StatusSeeOther {
		t.Fatalf("generated Admin delete status=%d body=%s", deletedByAdmin.Code, deletedByAdmin.Body.String())
	}
	if response := readFor("tenant-a"); response.Code != http.StatusOK {
		t.Fatalf("existing tenant record was lost after deleting Admin-created record: status=%d body=%s", response.Code, response.Body.String())
	}
	missingCSRF := adminMutation("/admin/resources/article/new", url.Values{"id": {"csrf-denied"}, "title": {"Must not be created"}})
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("generated Admin accepted a CSRF-less create: status=%d body=%s", missingCSRF.Code, missingCSRF.Body.String())
	}
	if response := readResourceFor("tenant-a", "csrf-denied"); response.Code != http.StatusNotFound {
		t.Fatalf("CSRF-less mutation was persisted: status=%d body=%s", response.Code, response.Body.String())
	}

	adminRequest := httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil)
	adminRequest.Header.Set("X-Test-Tenant", "tenant-a")
	adminResponse := httptest.NewRecorder()
	mux.ServeHTTP(adminResponse, adminRequest)
	var document map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &document); err != nil || document["title"] != "External tenant record" {
		t.Fatalf("generated API response did not contain the created document: %s (%v)", created.Body.String(), err)
	}
	if adminResponse.Code != http.StatusOK || !strings.Contains(adminResponse.Body.String(), "External tenant record") {
		t.Fatalf("external Edition Admin did not read the generated record: status=%d body=%s", adminResponse.Code, adminResponse.Body.String())
	}
}
