package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestRoleStoreLifecycle(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "roles.sqlite"))
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

	store := RoleStore{DB: db, Dialect: database.SQLite, RequireTenantScope: true}
	ctxA := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-a", Slug: "tenant-a"})
	ctxB := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-b", Slug: "tenant-b"})

	roleA := auth.CustomRole{
		ID:          "moderator",
		Name:        "Content Moderator",
		Description: "Can moderate comments and view posts",
		Permissions: []auth.RolePermission{
			{Resource: "collection:comments", Action: "*"},
			{Resource: "collection:posts", Action: "read"},
		},
	}

	if err := store.SaveRole(ctxA, roleA); err != nil {
		t.Fatalf("SaveRole failed: %v", err)
	}

	// Verify GetRole in tenant A
	gotA, err := store.GetRole(ctxA, "moderator")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if gotA.Name != "Content Moderator" || len(gotA.Permissions) != 2 {
		t.Fatalf("unexpected role in tenant A: %+v", gotA)
	}

	// Verify Tenant Isolation: tenant B cannot see tenant A's role
	listB, err := store.ListRoles(ctxB)
	if err != nil {
		t.Fatalf("ListRoles tenant B failed: %v", err)
	}
	if len(listB) != 0 {
		t.Fatalf("expected 0 roles in tenant B, got %d", len(listB))
	}

	if _, err := store.GetRole(ctxB, "moderator"); err != auth.ErrRoleNotFound {
		t.Fatalf("expected ErrRoleNotFound in tenant B, got %v", err)
	}

	// Update role permissions in tenant A
	roleA.Permissions = []auth.RolePermission{
		{Resource: "collection:comments", Action: "delete"},
	}
	if err := store.SaveRole(ctxA, roleA); err != nil {
		t.Fatalf("SaveRole update failed: %v", err)
	}
	gotUpdated, err := store.GetRole(ctxA, "moderator")
	if err != nil {
		t.Fatalf("GetRole updated failed: %v", err)
	}
	if len(gotUpdated.Permissions) != 1 || gotUpdated.Permissions[0].Action != "delete" {
		t.Fatalf("unexpected updated permissions: %+v", gotUpdated.Permissions)
	}

	// Delete role in tenant A
	if err := store.DeleteRole(ctxA, "moderator"); err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	if _, err := store.GetRole(ctxA, "moderator"); err != auth.ErrRoleNotFound {
		t.Fatalf("expected ErrRoleNotFound after delete, got %v", err)
	}
}
