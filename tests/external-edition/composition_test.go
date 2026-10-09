package externaledition

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	generated "example.com/external-edition/generated"
	generatedarticle "example.com/external-edition/generated/article"
	generatedmediaarticle "example.com/external-edition/generated/media_article"
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

type externalRoleContextKey struct{}

func externalRole(ctx context.Context) string {
	role, _ := ctx.Value(externalRoleContextKey{}).(string)
	return role
}

type externalPermissionOverrideContextKey struct{}

func externalPermissionOverride(ctx context.Context) string {
	override, _ := ctx.Value(externalPermissionOverrideContextKey{}).(string)
	return override
}

type externalResourcePolicy struct{}

func (externalResourcePolicy) CanRead(ctx context.Context, _ string) bool {
	role := externalRole(ctx)
	return role == "admin" || role == "editor" || role == "viewer"
}
func externalCanWrite(ctx context.Context) bool {
	role := externalRole(ctx)
	return role == "admin" || role == "editor"
}
func (externalResourcePolicy) CanCreate(ctx context.Context) bool {
	return externalCanWrite(ctx) || externalRole(ctx) == "viewer" && externalPermissionOverride(ctx) == "allow-create"
}
func (externalResourcePolicy) CanUpdate(ctx context.Context, _ string) bool {
	return externalCanWrite(ctx) && externalPermissionOverride(ctx) != "deny-update"
}
func (externalResourcePolicy) CanDelete(ctx context.Context, _ string) bool {
	return externalCanWrite(ctx) && externalPermissionOverride(ctx) != "deny-delete"
}

func serveWithExternalRole(mux *http.ServeMux, w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), externalRoleContextKey{}, r.Header.Get("X-Test-Role"))
	ctx = context.WithValue(ctx, externalPermissionOverrideContextKey{}, r.Header.Get("X-Test-Permission-Override"))
	mux.ServeHTTP(w, r.WithContext(ctx))
}

func TestExternalEditionGeneratedTenantCRUDAndAdminUseAggregateHelpers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), generatedarticle.MigrationSQL); err != nil {
		t.Fatalf("apply generated SQLite migration: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), generatedmediaarticle.MigrationSQL); err != nil {
		t.Fatalf("apply generated media article SQLite migration: %v", err)
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
			if resourceID != "article" && resourceID != "media_article" {
				return "", false
			}
			tenant := r.Header.Get("X-Test-Tenant")
			return tenant, tenant != ""
		},
	}); err != nil {
		t.Fatalf("register generated routes: %v", err)
	}
	if err := generated.RegisterGeneratedAdmin(mux, adminstudio.GeneratedAdminOptions{
		API: mux, Authorize: func(r *http.Request) bool {
			role := r.Header.Get("X-Test-Role")
			return role == "admin" || role == "editor" || role == "viewer"
		},
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		ResourceAccess: func(r *http.Request, _ adminstudio.Collection, id string) adminstudio.ResourceAccess {
			role, override := r.Header.Get("X-Test-Role"), r.Header.Get("X-Test-Permission-Override")
			if role == "viewer" {
				return adminstudio.ResourceAccess{Create: override == "allow-create"}
			}
			return adminstudio.ResourceAccess{Create: id == "", Update: id != "" && override != "deny-update", Delete: id != "" && override != "deny-delete"}
		},
		CSRFToken:    func(*http.Request) string { return "edition-token" },
		ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "edition-token" },
		RenderPage: adminstudio.StudioPageRenderer(func(r *http.Request) adminstudio.PageIdentity {
			if r.Header.Get("X-Test-Role") == "editor" {
				return adminstudio.PageIdentity{UserID: "edition-editor", Email: "editor@example.test", Role: adminstudio.PageRoleEditor}
			}
			if r.Header.Get("X-Test-Role") == "viewer" {
				return adminstudio.PageIdentity{UserID: "edition-viewer", Email: "viewer@example.test", Role: adminstudio.PageRoleViewer}
			}
			return adminstudio.PageIdentity{UserID: "edition-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}); err != nil {
		t.Fatalf("register generated Admin: %v", err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), externalRoleContextKey{}, r.Header.Get("X-Test-Role"))
		ctx = context.WithValue(ctx, externalPermissionOverrideContextKey{}, r.Header.Get("X-Test-Permission-Override"))
		mux.ServeHTTP(w, r.WithContext(ctx))
	})

	create := httptest.NewRequest(http.MethodPost, "/api/v1/article", strings.NewReader(`{"id":"article-1","title":"External tenant record"}`))
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("X-Test-Tenant", "tenant-a")
	create.Header.Set("X-Test-Role", "admin")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("generated tenant create status=%d body=%s", created.Code, created.Body.String())
	}

	readResourceFor := func(tenant, id string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/article/"+id, nil)
		req.Header.Set("X-Test-Tenant", tenant)
		req.Header.Set("X-Test-Role", "admin")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	readFor := func(tenant string) *httptest.ResponseRecorder { return readResourceFor(tenant, "article-1") }
	if response := readFor("tenant-a"); response.Code != http.StatusOK {
		t.Fatalf("same-tenant read status=%d body=%s", response.Code, response.Body.String())
	}

	viewerCreateOverride := httptest.NewRequest(http.MethodPost, "/admin/resources/article/new", strings.NewReader("_csrf=edition-token&id=viewer-override-created&title=Explicitly+granted+record"))
	viewerCreateOverride.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	viewerCreateOverride.Header.Set("X-Test-Tenant", "tenant-a")
	viewerCreateOverride.Header.Set("X-Test-Role", "viewer")
	viewerCreateOverride.Header.Set("X-Test-Permission-Override", "allow-create")
	viewerCreateOverrideResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerCreateOverrideResponse, viewerCreateOverride)
	if viewerCreateOverrideResponse.Code != http.StatusSeeOther {
		t.Fatalf("explicit Viewer create override status=%d body=%s", viewerCreateOverrideResponse.Code, viewerCreateOverrideResponse.Body.String())
	}
	if response := readResourceFor("tenant-a", "viewer-override-created"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Explicitly granted record") {
		t.Fatalf("explicit Viewer create override did not persist through Edition API: status=%d body=%s", response.Code, response.Body.String())
	}
	viewerOverrideList := httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil)
	viewerOverrideList.Header.Set("X-Test-Tenant", "tenant-a")
	viewerOverrideList.Header.Set("X-Test-Role", "viewer")
	viewerOverrideList.Header.Set("X-Test-Permission-Override", "allow-create")
	viewerOverrideListResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerOverrideListResponse, viewerOverrideList)
	if viewerOverrideListResponse.Code != http.StatusOK || !strings.Contains(viewerOverrideListResponse.Body.String(), `href="/admin/resources/article/new"`) {
		t.Fatalf("explicit Viewer create override did not expose the create action: status=%d body=%s", viewerOverrideListResponse.Code, viewerOverrideListResponse.Body.String())
	}

	for _, deniedOverride := range []struct {
		role, override, path, body, unchanged string
	}{
		{"editor", "deny-update", "/admin/resources/article/article-1", "_csrf=edition-token&title=Must+remain+unchanged", "External tenant record"},
		{"admin", "deny-delete", "/admin/resources/article/article-1/delete", "_csrf=edition-token", "External tenant record"},
	} {
		request := httptest.NewRequest(http.MethodPost, deniedOverride.path, strings.NewReader(deniedOverride.body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("X-Test-Tenant", "tenant-a")
		request.Header.Set("X-Test-Role", deniedOverride.role)
		request.Header.Set("X-Test-Permission-Override", deniedOverride.override)
		response := httptest.NewRecorder()
		serveWithExternalRole(mux, response, request)
		if response.Code != http.StatusForbidden {
			t.Errorf("%s %s override status=%d body=%s", deniedOverride.role, deniedOverride.override, response.Code, response.Body.String())
		}
		detailRequest := httptest.NewRequest(http.MethodGet, "/admin/resources/article/article-1", nil)
		detailRequest.Header.Set("X-Test-Tenant", "tenant-a")
		detailRequest.Header.Set("X-Test-Role", deniedOverride.role)
		detailRequest.Header.Set("X-Test-Permission-Override", deniedOverride.override)
		detailResponse := httptest.NewRecorder()
		serveWithExternalRole(mux, detailResponse, detailRequest)
		if detailResponse.Code != http.StatusOK {
			t.Errorf("%s %s override detail status=%d body=%s", deniedOverride.role, deniedOverride.override, detailResponse.Code, detailResponse.Body.String())
		} else if deniedOverride.override == "deny-update" && strings.Contains(detailResponse.Body.String(), "<form") {
			t.Errorf("read-only Editor still saw a writable detail form: %s", detailResponse.Body.String())
		} else if deniedOverride.override == "deny-delete" && strings.Contains(detailResponse.Body.String(), "/delete") {
			t.Errorf("Admin delete-deny override still exposed the delete action: %s", detailResponse.Body.String())
		}
		if persisted := readFor("tenant-a"); persisted.Code != http.StatusOK || !strings.Contains(persisted.Body.String(), deniedOverride.unchanged) {
			t.Errorf("%s %s override changed persisted record: status=%d body=%s", deniedOverride.role, deniedOverride.override, persisted.Code, persisted.Body.String())
		}
	}

	for _, role := range []string{"", "unknown"} {
		role := role
		t.Run("unauthorized-role/"+map[string]string{"": "anonymous", "unknown": "unknown"}[role], func(t *testing.T) {
			for _, target := range []struct {
				method string
				path   string
				body   string
			}{
				{http.MethodGet, "/admin/resources", ""},
				{http.MethodGet, "/admin/resources/article", ""},
				{http.MethodGet, "/admin/resources/article/new", ""},
				{http.MethodGet, "/admin/resources/article/article-1", ""},
				{http.MethodPost, "/admin/resources/article/new", "_csrf=edition-token&id=unauthorized-created&title=Must+not+exist"},
				{http.MethodPost, "/admin/resources/article/article-1", "_csrf=edition-token&title=Must+not+change"},
				{http.MethodPost, "/admin/resources/article/article-1/delete", "_csrf=edition-token"},
				{http.MethodGet, "/api/v1/article", ""},
				{http.MethodGet, "/api/v1/article/article-1", ""},
				{http.MethodPost, "/api/v1/article", `{"id":"unauthorized-api-created","title":"Must not exist"}`},
				{http.MethodPut, "/api/v1/article/article-1", `{"title":"Must not change"}`},
				{http.MethodDelete, "/api/v1/article/article-1", ""},
			} {
				request := httptest.NewRequest(target.method, target.path, strings.NewReader(target.body))
				if strings.HasPrefix(target.path, "/admin/") {
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				} else if target.method != http.MethodGet {
					request.Header.Set("Content-Type", "application/json")
				}
				request.Header.Set("X-Test-Tenant", "tenant-a")
				request.Header.Set("X-Test-Role", role)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				wantStatus := http.StatusForbidden
				if target.method == http.MethodGet && target.path == "/api/v1/article/article-1" {
					wantStatus = http.StatusNotFound // Conceal whether an unreadable record exists.
				}
				if response.Code != wantStatus {
					t.Errorf("role %q %s %s status=%d, want %d", role, target.method, target.path, response.Code, wantStatus)
				}
			}
			if response := readResourceFor("tenant-a", "unauthorized-created"); response.Code != http.StatusNotFound {
				t.Errorf("role %q created an Admin record: status=%d", role, response.Code)
			}
			if response := readResourceFor("tenant-a", "unauthorized-api-created"); response.Code != http.StatusNotFound {
				t.Errorf("role %q created an API record: status=%d", role, response.Code)
			}
			if response := readFor("tenant-a"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "External tenant record") {
				t.Errorf("role %q changed the existing record: status=%d body=%s", role, response.Code, response.Body.String())
			}
		})
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
		request.Header.Set("X-Test-Role", "admin")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s API record status=%d body=%s", record.tenant, response.Code, response.Body.String())
		}
	}
	adminList := func(tenant string) string {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil)
		request.Header.Set("X-Test-Tenant", tenant)
		request.Header.Set("X-Test-Role", "admin")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
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
	viewerList.Header.Set("X-Test-Tenant", "tenant-a")
	viewerListResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerListResponse, viewerList)
	if viewerListResponse.Code != http.StatusOK {
		t.Fatalf("generated Admin viewer list status=%d", viewerListResponse.Code)
	}
	if !strings.Contains(viewerListResponse.Body.String(), `data-studio-role="viewer"`) {
		t.Fatal("generated Admin shell did not render the viewer presentation role")
	}
	if strings.Contains(viewerListResponse.Body.String(), `href="/admin/resources/article/new"`) {
		t.Fatal("generated Admin viewer list exposed a create action")
	}
	if !strings.Contains(viewerListResponse.Body.String(), "External tenant record") {
		t.Fatal("generated Admin viewer list did not show the readable tenant record")
	}
	viewerNew := httptest.NewRequest(http.MethodGet, "/admin/resources/article/new", nil)
	viewerNew.Header.Set("X-Test-Role", "viewer")
	viewerNew.Header.Set("X-Test-Tenant", "tenant-a")
	viewerNewResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerNewResponse, viewerNew)
	if viewerNewResponse.Code != http.StatusForbidden {
		t.Fatalf("generated Admin allowed a viewer to open the create form directly: status=%d body=%s", viewerNewResponse.Code, viewerNewResponse.Body.String())
	}
	viewerDetail := httptest.NewRequest(http.MethodGet, "/admin/resources/article/article-1", nil)
	viewerDetail.Header.Set("X-Test-Role", "viewer")
	viewerDetail.Header.Set("X-Test-Tenant", "tenant-a")
	viewerDetailResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerDetailResponse, viewerDetail)
	if viewerDetailResponse.Code != http.StatusOK || strings.Contains(viewerDetailResponse.Body.String(), "<form") || !strings.Contains(viewerDetailResponse.Body.String(), "External tenant record") {
		t.Fatalf("generated Admin viewer detail was not read-only: status=%d", viewerDetailResponse.Code)
	}
	viewerUpdate := httptest.NewRequest(http.MethodPost, "/admin/resources/article/article-1", strings.NewReader("_csrf=edition-token&title=Must+remain+unchanged"))
	viewerUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	viewerUpdate.Header.Set("X-Test-Role", "viewer")
	viewerUpdate.Header.Set("X-Test-Tenant", "tenant-a")
	viewerUpdateResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerUpdateResponse, viewerUpdate)
	if viewerUpdateResponse.Code != http.StatusForbidden {
		t.Fatalf("generated Admin allowed a viewer update: status=%d", viewerUpdateResponse.Code)
	}
	viewerDelete := httptest.NewRequest(http.MethodPost, "/admin/resources/article/article-1/delete", strings.NewReader("_csrf=edition-token"))
	viewerDelete.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	viewerDelete.Header.Set("X-Test-Role", "viewer")
	viewerDelete.Header.Set("X-Test-Tenant", "tenant-a")
	viewerDeleteResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerDeleteResponse, viewerDelete)
	if viewerDeleteResponse.Code != http.StatusForbidden {
		t.Fatalf("generated Admin allowed a viewer delete: status=%d", viewerDeleteResponse.Code)
	}
	if response := readResourceFor("tenant-a", "article-1"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "External tenant record") {
		t.Fatalf("viewer update changed persisted data: status=%d body=%s", response.Code, response.Body.String())
	}
	editorList := httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil)
	editorList.Header.Set("X-Test-Role", "editor")
	editorList.Header.Set("X-Test-Tenant", "tenant-a")
	editorListResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, editorListResponse, editorList)
	if editorListResponse.Code != http.StatusOK || !strings.Contains(editorListResponse.Body.String(), `data-studio-role="editor"`) {
		t.Fatalf("generated Admin did not render the authenticated editor role: status=%d body=%s", editorListResponse.Code, editorListResponse.Body.String())
	}
	viewerMutation := httptest.NewRequest(http.MethodPost, "/admin/resources/article/new", strings.NewReader("_csrf=edition-token&id=viewer-created&title=Must+be+denied"))
	viewerMutation.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	viewerMutation.Header.Set("X-Test-Role", "viewer")
	viewerMutation.Header.Set("X-Test-Tenant", "tenant-a")
	viewerMutationResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerMutationResponse, viewerMutation)
	if viewerMutationResponse.Code != http.StatusForbidden {
		t.Fatalf("generated Admin allowed a viewer mutation: status=%d body=%s", viewerMutationResponse.Code, viewerMutationResponse.Body.String())
	}
	if response := readResourceFor("tenant-a", "viewer-created"); response.Code != http.StatusNotFound {
		t.Fatalf("viewer mutation was persisted: status=%d body=%s", response.Code, response.Body.String())
	}
	viewerAPIWrite := httptest.NewRequest(http.MethodPost, "/api/v1/article", strings.NewReader(`{"id":"viewer-api-created","title":"Must be denied"}`))
	viewerAPIWrite.Header.Set("Content-Type", "application/json")
	viewerAPIWrite.Header.Set("X-Test-Tenant", "tenant-a")
	viewerAPIWrite.Header.Set("X-Test-Role", "viewer")
	viewerAPIWriteResponse := httptest.NewRecorder()
	serveWithExternalRole(mux, viewerAPIWriteResponse, viewerAPIWrite)
	if viewerAPIWriteResponse.Code != http.StatusForbidden {
		t.Fatalf("generated API allowed viewer write: status=%d body=%s", viewerAPIWriteResponse.Code, viewerAPIWriteResponse.Body.String())
	}
	if response := readResourceFor("tenant-a", "viewer-api-created"); response.Code != http.StatusNotFound {
		t.Fatalf("viewer API mutation was persisted: status=%d body=%s", response.Code, response.Body.String())
	}
	for _, mutation := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPut, "/api/v1/article/article-1", `{"title":"Viewer API update denied"}`},
		{http.MethodDelete, "/api/v1/article/article-1", ""},
	} {
		request := httptest.NewRequest(mutation.method, mutation.path, strings.NewReader(mutation.body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Test-Tenant", "tenant-a")
		request.Header.Set("X-Test-Role", "viewer")
		response := httptest.NewRecorder()
		serveWithExternalRole(mux, response, request)
		if response.Code != http.StatusForbidden {
			t.Errorf("viewer API %s %s status=%d, want 403", mutation.method, mutation.path, response.Code)
		}
	}
	if response := readResourceFor("tenant-a", "article-1"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "External tenant record") {
		t.Fatalf("viewer API update/delete changed persisted data: status=%d body=%s", response.Code, response.Body.String())
	}

	adminMutation := func(path string, values url.Values) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("X-Test-Role", "admin")
		request.Header.Set("X-Test-Tenant", "tenant-a")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	editorMutation := func(path string, values url.Values) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("X-Test-Role", "editor")
		request.Header.Set("X-Test-Tenant", "tenant-a")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	newForm := httptest.NewRequest(http.MethodGet, "/admin/resources/article/new", nil)
	newForm.Header.Set("X-Test-Tenant", "tenant-a")
	newForm.Header.Set("X-Test-Role", "admin")
	newFormResponse := httptest.NewRecorder()
	handler.ServeHTTP(newFormResponse, newForm)
	if newFormResponse.Code != http.StatusOK || !strings.Contains(newFormResponse.Body.String(), `name="_csrf" value="edition-token"`) {
		t.Fatalf("generated Admin create form did not render a CSRF token: status=%d body=%s", newFormResponse.Code, newFormResponse.Body.String())
	}
	invalidTitle := adminMutation("/admin/resources/article/new", url.Values{"_csrf": {"edition-token"}, "id": {"blank-title"}, "title": {"   "}})
	if invalidTitle.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidTitle.Body.String(), "Bu alanları kontrol edin") || !strings.Contains(invalidTitle.Body.String(), "Bu alan zorunludur.") {
		t.Fatalf("whitespace-only required title did not render field feedback: status=%d body=%s", invalidTitle.Code, invalidTitle.Body.String())
	}
	if response := readResourceFor("tenant-a", "blank-title"); response.Code != http.StatusNotFound {
		t.Fatalf("invalid whitespace-only title was persisted: status=%d body=%s", response.Code, response.Body.String())
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
	invalidCSRF := adminMutation("/admin/resources/article/new", url.Values{"_csrf": {"forged-token"}, "id": {"csrf-forged"}, "title": {"Must not be created"}})
	if invalidCSRF.Code != http.StatusForbidden {
		t.Fatalf("generated Admin accepted an invalid CSRF token: status=%d body=%s", invalidCSRF.Code, invalidCSRF.Body.String())
	}
	if response := readResourceFor("tenant-a", "csrf-forged"); response.Code != http.StatusNotFound {
		t.Fatalf("invalid-CSRF create was persisted: status=%d body=%s", response.Code, response.Body.String())
	}
	csrfTarget := adminMutation("/admin/resources/article/new", url.Values{"_csrf": {"edition-token"}, "id": {"csrf-target"}, "title": {"Protected title"}})
	if csrfTarget.Code != http.StatusSeeOther {
		t.Fatalf("could not create CSRF mutation target: status=%d body=%s", csrfTarget.Code, csrfTarget.Body.String())
	}
	invalidCSRFUpdate := adminMutation("/admin/resources/article/csrf-target", url.Values{"_csrf": {"forged-token"}, "title": {"Forged update"}})
	if invalidCSRFUpdate.Code != http.StatusForbidden {
		t.Fatalf("generated Admin accepted an invalid CSRF update: status=%d body=%s", invalidCSRFUpdate.Code, invalidCSRFUpdate.Body.String())
	}
	if response := readResourceFor("tenant-a", "csrf-target"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Protected title") || strings.Contains(response.Body.String(), "Forged update") {
		t.Fatalf("invalid-CSRF update changed persisted state: status=%d body=%s", response.Code, response.Body.String())
	}
	invalidCSRFDelete := adminMutation("/admin/resources/article/csrf-target/delete", url.Values{"_csrf": {"forged-token"}})
	if invalidCSRFDelete.Code != http.StatusForbidden {
		t.Fatalf("generated Admin accepted an invalid CSRF delete: status=%d body=%s", invalidCSRFDelete.Code, invalidCSRFDelete.Body.String())
	}
	if response := readResourceFor("tenant-a", "csrf-target"); response.Code != http.StatusOK {
		t.Fatalf("invalid-CSRF delete removed the record: status=%d body=%s", response.Code, response.Body.String())
	}
	editorCreate := editorMutation("/admin/resources/article/new", url.Values{"_csrf": {"edition-token"}, "id": {"editor-created"}, "title": {"Created by editor"}})
	if editorCreate.Code != http.StatusSeeOther {
		t.Fatalf("editor generated Admin create status=%d body=%s", editorCreate.Code, editorCreate.Body.String())
	}
	editorUpdate := editorMutation("/admin/resources/article/editor-created", url.Values{"_csrf": {"edition-token"}, "title": {"Updated by editor"}})
	if editorUpdate.Code != http.StatusSeeOther {
		t.Fatalf("editor generated Admin update status=%d body=%s", editorUpdate.Code, editorUpdate.Body.String())
	}
	if response := readResourceFor("tenant-a", "editor-created"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Updated by editor") {
		t.Fatalf("editor CRUD was not persisted in its tenant: status=%d body=%s", response.Code, response.Body.String())
	}
	editorDelete := editorMutation("/admin/resources/article/editor-created/delete", url.Values{"_csrf": {"edition-token"}})
	if editorDelete.Code != http.StatusSeeOther {
		t.Fatalf("editor generated Admin delete status=%d body=%s", editorDelete.Code, editorDelete.Body.String())
	}
	if response := readResourceFor("tenant-a", "editor-created"); response.Code != http.StatusNotFound {
		t.Fatalf("editor delete left a persisted record: status=%d body=%s", response.Code, response.Body.String())
	}

	adminRequest := httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil)
	adminRequest.Header.Set("X-Test-Tenant", "tenant-a")
	adminRequest.Header.Set("X-Test-Role", "admin")
	adminResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminResponse, adminRequest)
	var document map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &document); err != nil || document["title"] != "External tenant record" {
		t.Fatalf("generated API response did not contain the created document: %s (%v)", created.Body.String(), err)
	}
	if adminResponse.Code != http.StatusOK || !strings.Contains(adminResponse.Body.String(), "External tenant record") {
		t.Fatalf("external Edition Admin did not read the generated record: status=%d body=%s", adminResponse.Code, adminResponse.Body.String())
	}
}

func TestExternalEditionGeneratedMediaFieldListsTenantAssetsAndPersistsReference(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, migration := range []string{generatedarticle.MigrationSQL, generatedmediaarticle.MigrationSQL} {
		if _, err := db.ExecContext(context.Background(), migration); err != nil {
			t.Fatalf("apply generated SQLite migration: %v", err)
		}
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /api/media", func(w http.ResponseWriter, r *http.Request) {
		tenant := r.Header.Get("X-Test-Tenant")
		if tenant == "" || (r.Header.Get("X-Test-Role") != "admin" && r.Header.Get("X-Test-Role") != "editor" && r.Header.Get("X-Test-Role") != "viewer") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]string{{
			"id": externalTenantMediaID(tenant), "status": "ready",
			"mime_type": "image/svg+xml", "original_filename": "edition-cover.svg",
		}})
	})
	if err := generated.RegisterGeneratedResources(apiMux, generated.GeneratedResourceOptions{
		DB: db, Dialect: "sqlite",
		PolicyFor: func(string) generated.GeneratedResourcePolicy { return externalResourcePolicy{} },
		ResolveTenant: func(resourceID string, r *http.Request) (string, bool) {
			if resourceID != "article" && resourceID != "media_article" {
				return "", false
			}
			tenant := r.Header.Get("X-Test-Tenant")
			return tenant, tenant != ""
		},
	}); err != nil {
		t.Fatalf("register generated APIs: %v", err)
	}
	api := enforceExternalMediaOwnership(apiMux)
	apiWithRole := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), externalRoleContextKey{}, r.Header.Get("X-Test-Role"))
		api.ServeHTTP(w, r.WithContext(ctx))
	})
	mux := http.NewServeMux()
	if err := generated.RegisterGeneratedAdmin(mux, adminstudio.GeneratedAdminOptions{
		API: apiWithRole,
		Authorize: func(r *http.Request) bool {
			return r.Header.Get("X-Test-Role") == "admin" || r.Header.Get("X-Test-Role") == "editor" || r.Header.Get("X-Test-Role") == "viewer"
		},
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		ResourceAccess: func(r *http.Request, _ adminstudio.Collection, id string) adminstudio.ResourceAccess {
			if r.Header.Get("X-Test-Role") == "viewer" {
				return adminstudio.ResourceAccess{}
			}
			return adminstudio.ResourceAccess{Create: id == "", Update: id != "", Delete: id != ""}
		},
		CSRFToken:    func(*http.Request) string { return "edition-token" },
		ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "edition-token" },
		RenderPage: adminstudio.StudioPageRenderer(func(r *http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "edition-editor", Email: "editor@example.test", Role: adminstudio.PageRoleEditor}
		}),
	}); err != nil {
		t.Fatalf("register generated Admin: %v", err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWithExternalRole(mux, w, r)
	})

	form := httptest.NewRequest(http.MethodGet, "/admin/resources/media_article/new", nil)
	form.Header.Set("X-Test-Tenant", "tenant-a")
	form.Header.Set("X-Test-Role", "editor")
	formResponse := httptest.NewRecorder()
	handler.ServeHTTP(formResponse, form)
	for _, marker := range []string{`data-media-target="cover_media_id"`, `name="cover_media_id"`, `id="admin-media-picker-modal"`} {
		if formResponse.Code != http.StatusOK || !strings.Contains(formResponse.Body.String(), marker) {
			t.Fatalf("generated media form missing %q: status=%d body=%s", marker, formResponse.Code, formResponse.Body.String())
		}
	}

	listAssets := func(tenant string) []map[string]string {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/api/media", nil)
		request.Header.Set("X-Test-Tenant", tenant)
		request.Header.Set("X-Test-Role", "editor")
		response := httptest.NewRecorder()
		apiWithRole.ServeHTTP(response, request)
		var assets []map[string]string
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &assets) != nil || len(assets) != 1 {
			t.Fatalf("%s media list status=%d body=%s", tenant, response.Code, response.Body.String())
		}
		return assets
	}
	assetA := listAssets("tenant-a")[0]["id"]
	assetB := listAssets("tenant-b")[0]["id"]
	if assetA == "" || assetA == assetB {
		t.Fatalf("tenant media IDs are not isolated: A=%q B=%q", assetA, assetB)
	}

	create := func(tenant, id, mediaID string) *httptest.ResponseRecorder {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{"id": id, "title": "Tenant media record", "cover_media_id": mediaID})
		request := httptest.NewRequest(http.MethodPost, generatedmediaarticle.MediaArticleRoute, strings.NewReader(string(payload)))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Test-Tenant", tenant)
		request.Header.Set("X-Test-Role", "editor")
		response := httptest.NewRecorder()
		apiWithRole.ServeHTTP(response, request)
		return response
	}
	created := create("tenant-a", "media-article-a", assetA)
	if created.Code != http.StatusCreated {
		t.Fatalf("tenant media create status=%d body=%s", created.Code, created.Body.String())
	}
	var record map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &record); err != nil || record["cover_media_id"] != assetA {
		t.Fatalf("media reference was not persisted: record=%v err=%v", record, err)
	}
	if denied := create("tenant-b", "media-article-forged", assetA); denied.Code != http.StatusUnprocessableEntity {
		t.Fatalf("cross-tenant media reference status=%d body=%s", denied.Code, denied.Body.String())
	}
	read := httptest.NewRequest(http.MethodGet, generatedmediaarticle.MediaArticleRoute+"/media-article-a", nil)
	read.Header.Set("X-Test-Tenant", "tenant-a")
	read.Header.Set("X-Test-Role", "editor")
	readResponse := httptest.NewRecorder()
	apiWithRole.ServeHTTP(readResponse, read)
	if readResponse.Code != http.StatusOK || !strings.Contains(readResponse.Body.String(), assetA) {
		t.Fatalf("same-tenant media reference read status=%d body=%s", readResponse.Code, readResponse.Body.String())
	}
	crossTenantRead := httptest.NewRequest(http.MethodGet, generatedmediaarticle.MediaArticleRoute+"/media-article-a", nil)
	crossTenantRead.Header.Set("X-Test-Tenant", "tenant-b")
	crossTenantRead.Header.Set("X-Test-Role", "editor")
	crossTenantResponse := httptest.NewRecorder()
	apiWithRole.ServeHTTP(crossTenantResponse, crossTenantRead)
	if crossTenantResponse.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant media article read status=%d body=%s", crossTenantResponse.Code, crossTenantResponse.Body.String())
	}
}

func externalTenantMediaID(tenant string) string {
	digest := sha256.Sum256([]byte(tenant))
	return "fixture-" + hex.EncodeToString(digest[:8])
}

func enforceExternalMediaOwnership(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, generatedmediaarticle.MediaArticleRoute) && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			var payload struct {
				CoverMediaID string `json:"cover_media_id"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if payload.CoverMediaID != "" && payload.CoverMediaID != externalTenantMediaID(r.Header.Get("X-Test-Tenant")) {
				http.Error(w, "media reference is not available to this tenant", http.StatusUnprocessableEntity)
				return
			}
			r.Body = io.NopCloser(strings.NewReader(string(body)))
		}
		next.ServeHTTP(w, r)
	})
}
