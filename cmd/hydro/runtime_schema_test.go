package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/auth"
	"github.com/hydrohq/hydrogo/internal/blueprints"
	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/collections/sqlstore"
	"github.com/hydrohq/hydrogo/internal/config"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"github.com/hydrohq/hydrogo/migrations"
)

func TestRuntimeSchemaDefinitionAPI(t *testing.T) {
	tempDir := t.TempDir()
	contentTypesDir := filepath.Join(tempDir, "content-types")
	if err := os.MkdirAll(contentTypesDir, 0o750); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tempDir, "runtime_schema.sqlite")
	db, err := database.Open(context.Background(), database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	colService := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite}}

	app := &serveApp{
		cfg: config.Config{
			ContentTypesDir: contentTypesDir,
			Database:        config.Database{Dialect: database.SQLite},
		},
		db:                db,
		collectionService: colService,
		contentTypes:      []contenttypes.Definition{},
	}

	adminUser := &auth.User{
		ID:     "admin-1",
		Role:   auth.RoleAdmin,
		Active: true,
	}

	mux := http.NewServeMux()
	for _, route := range app.contentTypeAdminRoutes() {
		r := route
		mux.HandleFunc(r.Pattern, func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(auth.WithUser(req.Context(), adminUser))
			r.Handler.ServeHTTP(w, req)
		})
	}

	// 1. Create collection via POST /api/v1/admin/content-types
	createBody := `{
		"id": "leads",
		"label": "Leads",
		"public": true,
		"fields": [
			{"id": "name", "label": "Name", "kind": "text", "required": true},
			{"id": "score", "label": "Score", "kind": "number"}
		]
	}`

	req := httptest.NewRequest("POST", "/api/v1/admin/content-types", strings.NewReader(createBody))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was written to disk
	yamlPath := filepath.Join(contentTypesDir, "leads.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		t.Fatalf("expected leads.yaml on disk: %v", err)
	}

	// Verify in-memory definition updated
	if len(app.contentTypes) != 1 || app.contentTypes[0].ID != "leads" {
		t.Fatalf("expected app.contentTypes to contain 'leads', got: %+v", app.contentTypes)
	}

	// Verify public collection API immediately recognizes 'leads' without restart
	_, publicRoutes, err := contentTypeRoutes(
		context.Background(),
		contentTypesDir,
		colService,
		nil,
		contenttypes.Locales{},
		nil,
		nil,
		nil,
		nil,
		func(r *http.Request) contenttypes.Principal {
			return contenttypes.Principal{Anonymous: false, Role: "admin", IsAdmin: true}
		},
		func() []contenttypes.Definition {
			return app.contentTypes
		},
	)
	if err != nil {
		t.Fatalf("contentTypeRoutes failed: %v", err)
	}

	publicMux := http.NewServeMux()
	for _, route := range publicRoutes {
		publicMux.Handle(route.Pattern, route.Handler)
	}

	// Create a record in the new collection
	createRecReq := httptest.NewRequest("POST", "/api/v1/content/leads", strings.NewReader(`{"name":"Jane Doe","score":95}`))
	createRecReq.SetPathValue("type", "leads")
	wRec := httptest.NewRecorder()
	publicMux.ServeHTTP(wRec, createRecReq)

	if wRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from public API, got %d: %s", wRec.Code, wRec.Body.String())
	}

	// 2. Update collection via PUT /api/v1/admin/content-types/leads
	updateBody := `{
		"id": "leads",
		"label": "Prospective Leads",
		"public": true,
		"fields": [
			{"id": "name", "label": "Name", "kind": "text", "required": true},
			{"id": "score", "label": "Score", "kind": "number"},
			{"id": "email", "label": "Email", "kind": "text"}
		]
	}`

	updateReq := httptest.NewRequest("PUT", "/api/v1/admin/content-types/leads", strings.NewReader(updateBody))
	updateReq.SetPathValue("id", "leads")
	wUpdate := httptest.NewRecorder()
	mux.ServeHTTP(wUpdate, updateReq)

	if wUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from update, got %d: %s", wUpdate.Code, wUpdate.Body.String())
	}

	if app.contentTypes[0].Label != "Prospective Leads" {
		t.Fatalf("expected updated label 'Prospective Leads', got %s", app.contentTypes[0].Label)
	}

	// 3. Delete collection via DELETE /api/v1/admin/content-types/leads
	delReq := httptest.NewRequest("DELETE", "/api/v1/admin/content-types/leads", nil)
	delReq.SetPathValue("id", "leads")
	wDel := httptest.NewRecorder()
	mux.ServeHTTP(wDel, delReq)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from delete, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Verify file was deleted from disk
	if _, err := os.Stat(yamlPath); !os.IsNotExist(err) {
		t.Fatalf("expected leads.yaml to be removed from disk, err: %v", err)
	}

	// Verify in-memory definition is removed
	if len(app.contentTypes) != 0 {
		t.Fatalf("expected 0 contentTypes after delete, got %d", len(app.contentTypes))
	}
}

func TestRuntimeSchemaTenantIsolationAndDynamicRegistration(t *testing.T) {
	tempDir := t.TempDir()
	contentTypesDir := filepath.Join(tempDir, "content-types")
	if err := os.MkdirAll(contentTypesDir, 0o750); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tempDir, "tenant_runtime_schema.sqlite")
	db, err := database.Open(context.Background(), database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	colService := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true}}
	resolver := tenant.NewHostResolver(nil, true)

	app := &serveApp{
		cfg: config.Config{
			ContentTypesDir: contentTypesDir,
			Database:        config.Database{Dialect: database.SQLite},
		},
		db:                 db,
		requireTenantScope: true,
		tenantResolver:     &resolver,
		collectionService:  colService,
		contentTypes:       []contenttypes.Definition{},
	}

	scopeA, _ := tenant.NewScope("tenant-a", "tenant-a")
	scopeB, _ := tenant.NewScope("tenant-b", "tenant-b")

	if err := app.registerTenantScope(context.Background(), "alpha.example.test", scopeA); err != nil {
		t.Fatalf("registerTenantScope A: %v", err)
	}
	if err := app.registerTenantScope(context.Background(), "beta.example.test", scopeB); err != nil {
		t.Fatalf("registerTenantScope B: %v", err)
	}
	if !resolver.Matches("alpha.example.test") || !resolver.Matches("beta.example.test") {
		t.Fatal("expected dynamic host registration to update HostResolver immediately")
	}

	adminUser := &auth.User{ID: "admin-1", Role: auth.RoleAdmin, Active: true}
	mux := http.NewServeMux()
	for _, route := range app.contentTypeAdminRoutes() {
		r := route
		mux.HandleFunc(r.Pattern, func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(auth.WithUser(req.Context(), adminUser))
			r.Handler.ServeHTTP(w, req)
		})
	}
	handler := resolver.Middleware(mux)

	// Tenant A defines a custom content type "appointments".
	createBody := `{"id":"appointments","label":"Appointments","public":true,"fields":[{"id":"client","label":"Client","kind":"text","required":true}]}`
	reqA := httptest.NewRequest("POST", "https://alpha.example.test/api/v1/admin/content-types", strings.NewReader(createBody))
	reqA.Host = "alpha.example.test"
	wA := httptest.NewRecorder()
	handler.ServeHTTP(wA, reqA)
	if wA.Code != http.StatusCreated {
		t.Fatalf("tenant A create content-type: %d %s", wA.Code, wA.Body.String())
	}

	// Verify file lives under tenants/tenant-a/appointments.yaml, NOT global root.
	tenantAYAML := filepath.Join(contentTypesDir, "tenants", "tenant-a", "appointments.yaml")
	if _, err := os.Stat(tenantAYAML); err != nil {
		t.Fatalf("expected %s to exist: %v", tenantAYAML, err)
	}
	if _, err := os.Stat(filepath.Join(contentTypesDir, "appointments.yaml")); !os.IsNotExist(err) {
		t.Fatal("tenant A content type must not pollute global content-types directory")
	}

	// Tenant B must NOT see Tenant A's content type.
	defsA := app.contentTypesForContext(tenant.WithScope(context.Background(), scopeA))
	defsB := app.contentTypesForContext(tenant.WithScope(context.Background(), scopeB))
	if len(defsA) != 1 || defsA[0].ID != "appointments" {
		t.Fatalf("expected tenant A to see appointments, got %+v", defsA)
	}
	if len(defsB) != 0 {
		t.Fatalf("expected tenant B to see 0 content types, got %+v", defsB)
	}
}

func TestTenantBlueprintHydrationAndContentTypeMutation(t *testing.T) {
	tempDir := t.TempDir()
	contentTypesDir := filepath.Join(tempDir, "content-types")
	if err := os.MkdirAll(contentTypesDir, 0o750); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tempDir, "blueprint_hydration.sqlite")
	db, err := database.Open(context.Background(), database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	colService := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true}}
	resolver := tenant.NewHostResolver(nil, false)
	scope, err := tenant.NewScope("corp-tenant", "corp-tenant")
	if err != nil {
		t.Fatal(err)
	}
	if err := blueprints.EnsureArchetypeSchemas(context.Background(), colService, scope, blueprints.ArchetypeCorporate); err != nil {
		t.Fatalf("EnsureArchetypeSchemas: %v", err)
	}

	app := &serveApp{
		cfg: config.Config{
			ContentTypesDir: contentTypesDir,
			Database:        config.Database{Dialect: database.SQLite},
		},
		db:                 db,
		requireTenantScope: true,
		tenantResolver:     &resolver,
		collectionService:  colService,
		contentTypes:       []contenttypes.Definition{},
	}
	if err := app.registerTenantScope(context.Background(), "corp.example.test", scope); err != nil {
		t.Fatalf("registerTenantScope: %v", err)
	}
	app.hydrateTenantBlueprint(scope, blueprints.ArchetypeCorporate)

	scopedCtx := tenant.WithScope(context.Background(), scope)
	initialDefs := app.contentTypesForContext(scopedCtx)
	if len(initialDefs) < 5 {
		t.Fatalf("expected hydrated corporate blueprint defs, got %d", len(initialDefs))
	}

	adminUser := &auth.User{ID: "admin-corp", Role: auth.RoleAdmin, Active: true}
	mux := http.NewServeMux()
	for _, route := range app.contentTypeAdminRoutes() {
		r := route
		mux.HandleFunc(r.Pattern, func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(auth.WithUser(req.Context(), adminUser))
			r.Handler.ServeHTTP(w, req)
		})
	}
	handler := resolver.Middleware(mux)

	// 1. Update one blueprint type (projects) and ensure sibling types are preserved.
	updateBody := `{"id":"projects","label":"Projeler","group":"Kurumsal Yönetim","public":true,"title_field":"title","fields":[{"id":"title","label":"Proje Adı","kind":"text","required":true,"list":true},{"id":"category","label":"Kategori","kind":"select","options":["Fintech","Enerji"],"list":true,"filterable":true}]}`
	reqPut := httptest.NewRequest(http.MethodPut, "https://corp.example.test/api/v1/admin/content-types/projects", strings.NewReader(updateBody))
	reqPut.Host = "corp.example.test"
	wPut := httptest.NewRecorder()
	handler.ServeHTTP(wPut, reqPut)
	if wPut.Code != http.StatusOK {
		t.Fatalf("PUT projects: %d %s", wPut.Code, wPut.Body.String())
	}

	afterUpdate := app.contentTypesForContext(scopedCtx)
	if len(afterUpdate) != len(initialDefs) {
		t.Fatalf("expected sibling blueprint defs preserved (%d), got %d", len(initialDefs), len(afterUpdate))
	}
	var hasSiteSettings, hasUpdatedOption bool
	for _, d := range afterUpdate {
		if d.ID == "site_settings" && d.Singleton {
			hasSiteSettings = true
		}
		if d.ID == "projects" {
			for _, f := range d.Fields {
				if f.ID == "category" && len(f.Options) == 2 && f.Options[1] == "Enerji" {
					hasUpdatedOption = true
				}
			}
		}
	}
	if !hasSiteSettings || !hasUpdatedOption {
		t.Fatalf("expected singleton site_settings=%v and updated projects options=%v", hasSiteSettings, hasUpdatedOption)
	}

	// 2. Delete an added custom content type and verify it is removed from memory + disk.
	createBody := `{"id":"custom_notes","label":"Notlar","fields":[{"id":"title","label":"Başlık","kind":"text","required":true}]}`
	reqPost := httptest.NewRequest(http.MethodPost, "https://corp.example.test/api/v1/admin/content-types", strings.NewReader(createBody))
	reqPost.Host = "corp.example.test"
	wPost := httptest.NewRecorder()
	handler.ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusCreated {
		t.Fatalf("POST custom_notes: %d %s", wPost.Code, wPost.Body.String())
	}

	reqDel := httptest.NewRequest(http.MethodDelete, "https://corp.example.test/api/v1/admin/content-types/custom_notes", nil)
	reqDel.Host = "corp.example.test"
	wDel := httptest.NewRecorder()
	handler.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("DELETE custom_notes: %d %s", wDel.Code, wDel.Body.String())
	}
	for _, d := range app.contentTypesForContext(scopedCtx) {
		if d.ID == "custom_notes" {
			t.Fatal("custom_notes remained in contentTypesForContext after deletion")
		}
	}

	// 3. A fresh serveApp instance hydrates blueprint + disk overrides cleanly.
	app2 := &serveApp{
		cfg:                app.cfg,
		db:                 db,
		requireTenantScope: true,
		collectionService:  colService,
	}
	app2.hydrateTenantBlueprint(scope, blueprints.ArchetypeCorporate)
	reloaded := app2.contentTypesForContext(scopedCtx)
	var reloadedEnerji bool
	for _, d := range reloaded {
		if d.ID == "projects" {
			for _, f := range d.Fields {
				if f.ID == "category" && len(f.Options) == 2 && f.Options[1] == "Enerji" {
					reloadedEnerji = true
				}
			}
		}
	}
	if !reloadedEnerji {
		t.Fatal("fresh serveApp did not merge tenant YAML override onto blueprint definitions")
	}
}
