package shell

import (
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

// NavItem is one sidebar link and the conditions under which it is shown.
type NavItem struct {
	ID    string
	Href  string
	Label Text
	Icon  string
	// Permission, when set, must be among the user's effective permissions.
	Permission string
	// Roles, when set, lists the only roles that see the item.
	Roles []auth.Role
	// Module, when set, must be enabled.
	Module string
	// Trash marks the recycle-bin link, shown only when View.Trash is set.
	Trash bool
	// ActiveFor lists the View.Active keys that highlight this item.
	ActiveFor []string
}

// NavSection is a labelled group of sidebar links.
type NavSection struct {
	Label Text
	Items []NavItem
}

var adminOnly = []auth.Role{auth.RoleAdmin}

// Navigation is the full Admin Studio sidebar. Every role sees the subset its
// permissions allow; nothing else decides what the sidebar contains.
var Navigation = []NavSection{
	{Label: Text{"GENEL", "GENERAL"}, Items: []NavItem{
		// Viewers have no dashboard: /admin answers 403 for them.
		{ID: "dashboard", Href: "/admin", Label: Text{"Dashboard", "Dashboard"}, Icon: "dashboard", Roles: []auth.Role{auth.RoleAdmin, auth.RoleEditor}, ActiveFor: []string{"dashboard"}},
		{ID: "reports", Href: "/admin/reports", Label: Text{"Raporlar", "Reports"}, Icon: "reports", Roles: adminOnly, ActiveFor: []string{"reports"}},
	}},
	{Label: Text{"İÇERİK", "CONTENT"}, Items: []NavItem{
		{ID: "news", Href: "/admin/news", Label: Text{"Haberler", "News"}, Icon: "news", Permission: auth.PermissionContentRead, Module: "news", ActiveFor: []string{"news"}},
		{ID: "pages", Href: "/admin/pages", Label: Text{"Sayfalar", "Pages"}, Icon: "pages", Permission: auth.PermissionContentRead, Module: "pages", ActiveFor: []string{"pages"}},
		{ID: "content-types", Href: "/admin/resources", Label: Text{"İçerik Tipleri", "Content types"}, Icon: "content-types", Roles: adminOnly, Module: "collections", ActiveFor: []string{"content-types"}},
	}},
	{Label: Text{"KULLANICILAR", "USERS"}, Items: []NavItem{
		{ID: "members", Href: "/admin/members", Label: Text{"Üyeler", "Members"}, Icon: "members", Permission: auth.PermissionMembersManage, Module: "members", ActiveFor: []string{"members"}},
		{ID: "users", Href: "/admin/users", Label: Text{"Kullanıcılar", "Users"}, Icon: "users", Permission: auth.PermissionUsersManage, Roles: adminOnly, ActiveFor: []string{"users"}},
	}},
	{Label: Text{"SİSTEM", "SYSTEM"}, Items: []NavItem{
		{ID: "media", Href: "/admin/media", Label: Text{"Medya", "Media"}, Icon: "media", Permission: auth.PermissionMediaManage, Module: "media", ActiveFor: []string{"media"}},
		{ID: "site", Href: "/admin/site", Label: Text{"Site ayarları", "Site settings"}, Icon: "site", Permission: auth.PermissionSiteManage, Module: "site", ActiveFor: []string{"site"}},
		{ID: "developer", Href: "/admin/developer", Label: Text{"Geliştirici & API", "Developer & API"}, Icon: "developer", Roles: adminOnly, ActiveFor: []string{"developer", "resources"}},
		{ID: "trash", Href: "/admin/trash", Label: Text{"Çöp kutusu", "Trash"}, Icon: "trash", Trash: true, ActiveFor: []string{"trash"}},
	}},
}

// VisibleSection is a section with the items the current user may see.
type VisibleSection struct {
	Label string
	Items []VisibleItem
}

// VisibleItem is a sidebar link ready to render.
type VisibleItem struct {
	ID       string
	Href     string
	Label    string
	Icon     string
	Active   bool
	Children []VisibleItem
}

// Sidebar filters Navigation for the view's user, locale and modules. Empty
// sections are dropped.
func (v View) Sidebar() []VisibleSection {
	sections := make([]VisibleSection, 0, len(Navigation))
	for _, section := range Navigation {
		visible := VisibleSection{Label: v.T(section.Label)}
		for _, item := range section.Items {
			if !v.allows(item) {
				continue
			}
			entry := VisibleItem{ID: item.ID, Href: item.Href, Label: v.T(item.Label), Icon: item.Icon, Active: contains(item.ActiveFor, v.Active)}
			if item.ID == "content-types" {
				entry.Children = v.contentTypeLinks()
				for _, child := range entry.Children {
					if child.Active {
						entry.Active = false
					}
				}
			}
			visible.Items = append(visible.Items, entry)
		}
		if len(visible.Items) > 0 {
			sections = append(sections, visible)
		}
	}
	return sections
}

func (v View) allows(item NavItem) bool {
	if item.Trash && !v.Trash {
		return false
	}
	if len(item.Roles) > 0 && !containsRole(item.Roles, v.User.Role) {
		return false
	}
	if item.Permission != "" && !v.User.Can(item.Permission) {
		return false
	}
	return v.moduleEnabled(item.Module)
}

func (v View) contentTypeLinks() []VisibleItem {
	links := make([]VisibleItem, 0, len(v.ContentTypes))
	for _, contentType := range v.ContentTypes {
		href := "/admin/resources/" + contentType.ID
		active := v.Path == href || strings.HasPrefix(v.Path, href+"/")
		links = append(links, VisibleItem{ID: "content-type-" + contentType.ID, Href: href, Label: contentType.Label, Active: active})
	}
	return links
}

// CanSeeSecurity reports whether the user menu links to security settings.
func (v View) CanSeeSecurity() bool {
	return v.User.Role == string(auth.RoleAdmin) && v.User.Can(auth.PermissionUsersManage)
}

// RoleLabel names the account type in the user menu.
func (v View) RoleLabel() string {
	switch auth.Role(v.User.Role) {
	case auth.RoleEditor:
		return v.T(Text{"Editör Hesabı", "Editor account"})
	case auth.RoleViewer:
		return v.T(Text{"İzleyici Hesabı", "Viewer account"})
	default:
		return v.T(Text{"Yönetici Hesabı", "Administrator account"})
	}
}

func navItemClass(item VisibleItem) string {
	class := "nav-item"
	if item.Icon == "" {
		class += " nav-subitem"
	}
	if item.Active {
		class += " is-active"
	}
	return class
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsRole(roles []auth.Role, role string) bool {
	for _, candidate := range roles {
		if string(candidate) == role {
			return true
		}
	}
	return false
}

// HomeHref is where the brand link goes: the first sidebar link the user may
// open (the dashboard for admins and editors, e.g. news for viewers), so the
// logo never leads to a page that answers 403.
func (v View) HomeHref() string {
	for _, section := range v.Sidebar() {
		for _, item := range section.Items {
			return item.Href
		}
	}
	return "/admin"
}
