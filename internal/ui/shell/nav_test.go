package shell

import (
	"slices"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func sidebarIDs(view View) []string {
	var ids []string
	for _, section := range view.Sidebar() {
		for _, item := range section.Items {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func permissionsOf(user *auth.User) []string {
	var allowed []string
	for _, permission := range auth.KnownPermissions {
		if auth.Can(user, permission) {
			allowed = append(allowed, permission)
		}
	}
	return allowed
}

func TestSidebarShowsEachRoleItsSubset(t *testing.T) {
	for _, tc := range []struct {
		role auth.Role
		want []string
	}{
		{auth.RoleAdmin, []string{"dashboard", "reports", "news", "pages", "content-types", "members", "users", "media", "site", "developer"}},
		{auth.RoleEditor, []string{"dashboard", "news", "pages", "media"}},
		{auth.RoleViewer, []string{"news", "pages"}},
	} {
		user := &auth.User{Role: tc.role, Active: true}
		view := View{User: User{Role: string(tc.role), Permissions: permissionsOf(user)}}
		if got := sidebarIDs(view); !slices.Equal(got, tc.want) {
			t.Errorf("%s sidebar = %v, want %v", tc.role, got, tc.want)
		}
	}
}

func TestSidebarHidesInactiveModulesAndEmptySections(t *testing.T) {
	user := &auth.User{Role: auth.RoleAdmin, Active: true}
	view := View{User: User{Role: "admin", Permissions: permissionsOf(user)}, Modules: map[string]bool{"pages": true}}
	got := sidebarIDs(view)
	for _, hidden := range []string{"news", "members", "media", "site", "content-types"} {
		if slices.Contains(got, hidden) {
			t.Errorf("sidebar shows %q although its module is off: %v", hidden, got)
		}
	}
	for _, section := range view.Sidebar() {
		if len(section.Items) == 0 {
			t.Errorf("empty section %q rendered", section.Label)
		}
	}
	if !slices.Contains(got, "users") || !slices.Contains(got, "developer") {
		t.Errorf("core admin entries must not depend on modules: %v", got)
	}
}

func TestSidebarListsContentTypesAndHighlightsTheCurrentOne(t *testing.T) {
	view := View{
		User:         User{Role: "admin"},
		Active:       "content-types",
		Path:         "/admin/resources/faq/42",
		ContentTypes: []ContentType{{ID: "faq", Label: "FAQ"}, {ID: "team", Label: "Team"}},
		Locale:       "en",
	}
	for _, section := range view.Sidebar() {
		for _, item := range section.Items {
			if item.ID != "content-types" {
				continue
			}
			if item.Label != "Content types" || item.Active {
				t.Fatalf("parent entry = %+v, want inactive English label while a child is current", item)
			}
			if len(item.Children) != 2 || !item.Children[0].Active || item.Children[1].Active || item.Children[0].Href != "/admin/resources/faq" {
				t.Fatalf("content type children = %+v", item.Children)
			}
			return
		}
	}
	t.Fatal("content types entry missing for admin")
}

func TestTrashLinkOnlyWhenEnabled(t *testing.T) {
	view := View{User: User{Role: "admin"}}
	if slices.Contains(sidebarIDs(view), "trash") {
		t.Fatal("trash shown without the trash routes")
	}
	view.Trash, view.Active = true, "trash"
	if !slices.Contains(sidebarIDs(view), "trash") {
		t.Fatal("trash missing when enabled")
	}
}

func TestHomeHrefPointsAtFirstAllowedPage(t *testing.T) {
	modules := map[string]bool{"news": true, "pages": true}
	admin := View{User: User{Role: "admin", Permissions: []string{auth.PermissionContentRead}}, Modules: modules}
	if got := admin.HomeHref(); got != "/admin" {
		t.Fatalf("admin home = %q", got)
	}
	viewer := View{User: User{Role: "viewer", Permissions: []string{auth.PermissionContentRead}}, Modules: modules}
	if got := viewer.HomeHref(); got == "/admin" {
		t.Fatalf("viewer home = %q, want a page the viewer may open", got)
	}
}
