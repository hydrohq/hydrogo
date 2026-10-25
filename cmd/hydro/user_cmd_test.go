package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestCreateUserMakesNonAdminAccountsAndIsRepeatable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "users.sqlite")
	base := []string{"--database-dialect", "sqlite", "--database-dsn", dbPath}
	var out, errb bytes.Buffer
	if err := run(append([]string{"migrate"}, base...), strings.NewReader(""), &out, &errb); err != nil {
		t.Fatalf("migrate: %v %s", err, errb.String())
	}
	user := func(args ...string) (string, error) {
		out.Reset()
		errb.Reset()
		err := run(append(append([]string{"create-user", "--password-stdin"}, args...), base...), strings.NewReader("Amber-River7!Cedar\n"), &out, &errb)
		return out.String(), err
	}

	got, err := user("--email", "Ayse@Example.test", "--permissions", "clinic.staff, clinic.reception")
	if err != nil || !strings.Contains(got, "Created user: ayse@example.test (role: viewer") || !strings.Contains(got, "clinic.staff, clinic.reception") {
		t.Fatalf("create: %q %v", got, err)
	}

	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := authstore.NewUserStore(db, database.SQLite)
	u, err := store.GetByEmail(ctx, "ayse@example.test")
	perms, permErr := store.LoadPermissionOverrides(ctx, u.ID)
	if err != nil || permErr != nil || u.Role != auth.RoleViewer || !perms["clinic.staff"] || !perms["clinic.reception"] {
		t.Fatalf("stored user: %+v %v perms=%v %v", u, err, perms, permErr)
	}

	// Running it again corrects the role and replaces the permissions.
	got, err = user("--email", "ayse@example.test", "--role", "editor", "--permissions", "clinic.manage")
	if err != nil || !strings.HasPrefix(got, "Updated user:") {
		t.Fatalf("update: %q %v", got, err)
	}
	u, _ = store.GetByEmail(ctx, "ayse@example.test")
	perms, _ = store.LoadPermissionOverrides(ctx, u.ID)
	if u.Role != auth.RoleEditor || perms["clinic.staff"] || !perms["clinic.manage"] {
		t.Fatalf("after update: %+v perms=%v", u, perms)
	}

	for name, args := range map[string][]string{
		"admin role is for create-admin": {"--email", "x@example.test", "--role", "admin"},
		"unknown role":                   {"--email", "x@example.test", "--role", "boss"},
		"no email":                       {},
	} {
		if _, err := user(args...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	// An administrator cannot be turned into a restricted user by accident.
	if err := run(append([]string{"create-admin", "--email", "root@example.test", "--password-stdin"}, base...), strings.NewReader("Amber-River7!Cedar\n"), &out, &errb); err != nil {
		t.Fatal(err)
	}
	if _, err := user("--email", "root@example.test"); err == nil || !strings.Contains(err.Error(), "administrator") {
		t.Errorf("administrator accounts must be refused: %v", err)
	}
}
