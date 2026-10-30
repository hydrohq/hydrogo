package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestAdminRolesAPIEndpoints(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "roles_api.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	schema := `
	CREATE TABLE hydro_roles (
		tenant_id TEXT NOT NULL DEFAULT '',
		id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		PRIMARY KEY (tenant_id, id)
	);
	CREATE TABLE hydro_role_permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id TEXT NOT NULL DEFAULT '',
		role_id TEXT NOT NULL,
		resource TEXT NOT NULL,
		action TEXT NOT NULL,
		FOREIGN KEY (tenant_id, role_id) REFERENCES hydro_roles(tenant_id, id) ON DELETE CASCADE
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}

	roleStore := authstore.RoleStore{DB: db, Dialect: database.SQLite}
	app := &serveApp{
		roleStore: roleStore,
	}

	adminUser := &auth.User{ID: "admin-1", Role: auth.RoleAdmin, Active: true}
	editorUser := &auth.User{ID: "editor-1", Role: auth.RoleEditor, Active: true}

	// 1. Non-admin forbidden
	req := adminAPIRequest("GET", "/api/v1/admin/roles", "", editorUser)
	rec := httptest.NewRecorder()
	app.adminRolesListHandler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	// 2. Admin list roles (empty)
	req = adminAPIRequest("GET", "/api/v1/admin/roles", "", adminUser)
	rec = httptest.NewRecorder()
	app.adminRolesListHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeEnvelope(t, rec)
	data, ok := body["data"].([]any)
	if !ok || len(data) != 0 {
		t.Fatalf("expected empty roles array, got %+v", body)
	}

	// 3. Create custom role
	createJSON := `{"id":"moderator","name":"Moderator","description":"Moderates content","permissions":[{"resource":"collection:posts","action":"read"},{"resource":"collection:comments","action":"*"}]}`
	req = adminAPIRequest("POST", "/api/v1/admin/roles", createJSON, adminUser)
	rec = httptest.NewRecorder()
	app.adminRoleSaveHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on create, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Get created role
	req = adminAPIRequest("GET", "/api/v1/admin/roles/moderator", "", adminUser)
	req.SetPathValue("id", "moderator")
	rec = httptest.NewRecorder()
	app.adminRoleGetHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d: %s", rec.Code, rec.Body.String())
	}
	body = decodeEnvelope(t, rec)
	roleData := body["data"].(map[string]any)
	if roleData["name"] != "Moderator" {
		t.Fatalf("unexpected role name: %v", roleData["name"])
	}

	// 5. Delete role
	req = adminAPIRequest("DELETE", "/api/v1/admin/roles/moderator", "", adminUser)
	req.SetPathValue("id", "moderator")
	rec = httptest.NewRecorder()
	app.adminRoleDeleteHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Verify role not found
	req = adminAPIRequest("GET", "/api/v1/admin/roles/moderator", "", adminUser)
	req.SetPathValue("id", "moderator")
	rec = httptest.NewRecorder()
	app.adminRoleGetHandler(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
