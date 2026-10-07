package auth

import (
	"context"
	"strings"
	"testing"
	"time"
)

type permissionTestStore struct {
	loaded map[string]bool
	saved  map[string]bool
}

func (s *permissionTestStore) LoadPermissionOverrides(context.Context, string) (map[string]bool, error) {
	return s.loaded, nil
}

func (s *permissionTestStore) ReplacePermissionOverrides(_ context.Context, _ string, overrides map[string]bool) error {
	s.saved = overrides
	return nil
}

func TestPermissionCatalogRequiresExplicitBoundedEditionNames(t *testing.T) {
	catalog, err := NewPermissionCatalog("article.read", "article.write")
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Contains(PermissionContentRead) || !catalog.Contains("article.read") {
		t.Fatalf("catalog omitted registered permission names: %#v", catalog.Names())
	}
	for _, invalid := range []string{"Article.read", " article.read", "article..read", "x", strings.Repeat("a", 130)} {
		if _, err := NewPermissionCatalog(invalid); err == nil {
			t.Errorf("invalid catalog entry %q was accepted", invalid)
		}
	}
	tooMany := make([]string, MaxPermissionCatalogEntries)
	// Use distinct valid names to exercise the hard entry cap.
	for i := range tooMany {
		tooMany[i] = "edition.permission:" + string(rune('a'+i%26)) + "." + string(rune('a'+i/26))
	}
	if _, err := NewPermissionCatalog(tooMany...); err == nil {
		t.Fatal("oversized permission catalog was accepted")
	}
}

func TestPermissionCatalogAllowsOnlyRegisteredPermissionsAndExplicitOverrides(t *testing.T) {
	catalog, err := NewPermissionCatalog("article.read")
	if err != nil {
		t.Fatal(err)
	}
	viewer := &User{Role: RoleViewer, Active: true}
	if catalog.Allows(viewer, "article.read") {
		t.Fatal("Edition permission unexpectedly inherited a role grant")
	}
	viewer.PermissionOverrides = map[string]bool{"article.read": true}
	if !catalog.Allows(viewer, "article.read") {
		t.Fatal("registered explicit allow override was ignored")
	}
	if catalog.Allows(&User{Role: RoleAdmin, Active: true}, "article.unknown") {
		t.Fatal("administrator was granted a permission outside the catalog")
	}
}

func TestPermissionCatalogRoleAndOverrideMatrix(t *testing.T) {
	catalog, err := NewPermissionCatalog("article.read")
	if err != nil {
		t.Fatal(err)
	}
	roleGrants := map[Role]map[string]bool{
		RoleAdmin:  {},
		RoleEditor: {PermissionContentRead: true, PermissionContentWrite: true, PermissionCommerceRead: true, PermissionMediaManage: true},
		RoleViewer: {PermissionContentRead: true, PermissionCommerceRead: true},
	}
	for _, role := range []Role{RoleAdmin, RoleEditor, RoleViewer} {
		for _, permission := range KnownPermissions {
			want := role == RoleAdmin || roleGrants[role][permission]
			t.Run(string(role)+"/"+permission, func(t *testing.T) {
				user := &User{Role: role, Active: true}
				if got := catalog.Allows(user, permission); got != want {
					t.Fatalf("default grant = %t, want %t", got, want)
				}
				user.PermissionOverrides = map[string]bool{permission: !want}
				if got := catalog.Allows(user, permission); got == want {
					t.Fatalf("explicit override did not change grant; got %t", got)
				}
				user.Active = false
				user.PermissionOverrides[permission] = true
				if catalog.Allows(user, permission) {
					t.Fatal("inactive user retained permission through override")
				}
			})
		}
	}
	for _, role := range []Role{RoleAdmin, RoleEditor, RoleViewer} {
		user := &User{Role: role, Active: true}
		if got, want := catalog.Allows(user, "article.read"), role == RoleAdmin; got != want {
			t.Errorf("%s Edition permission = %t, want %t", role, got, want)
		}
	}
}

func TestPermissionCatalogRejectsUnknownWritesAndFiltersRemovedPermissions(t *testing.T) {
	catalog, err := NewPermissionCatalog("article.read")
	if err != nil {
		t.Fatal(err)
	}
	store := &permissionTestStore{loaded: map[string]bool{"article.read": true, "removed.write": true}}
	service := NewAuthService(httpUserRepo{user: &User{ID: "u1", Active: true, Role: RoleViewer}}, &httpSessionRepo{session: &Session{ID: "s1", UserID: "u1", TokenHash: HashSessionToken("test-token"), ExpiresAt: time.Now().Add(time.Hour), LastSeenAt: time.Now()}}, nil, TestArgon2Params)
	service.SetPermissionStore(store)
	if err := service.SetPermissionCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	if err := service.ReplaceUserPermissions(context.Background(), "u1", map[string]bool{"removed.write": true}); err == nil {
		t.Fatal("unregistered permission override was accepted")
	}
	_, user, err := service.ValidateSession(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if len(user.PermissionOverrides) != 1 || !user.PermissionOverrides["article.read"] {
		t.Fatalf("active session retained unregistered grants: %#v", user.PermissionOverrides)
	}
	if err := service.ReplaceUserPermissions(context.Background(), "u1", map[string]bool{"article.read": true}); err != nil {
		t.Fatalf("registered permission override rejected: %v", err)
	}
}
