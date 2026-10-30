package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
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
