package ui

import (
	"context"
	"io"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/modules"
	"github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

// ModuleSidebar adapts the active module registry to the typed design system.
// It performs no authorization decision; callers must pass only modules that
// are enabled for the current Edition and whose routes are already protected.
func ModuleSidebar(registry *modules.Registry, activeRoute string) templ.Component {
	items := make([]components.SidebarItem, 0)
	if registry != nil {
		for _, item := range registry.AdminNavigation() {
			items = append(items, components.SidebarItem{Route: safeRoute(item.Route), Label: item.Label, Icon: item.Icon, Group: item.Group, Active: item.Route == activeRoute})
		}
	}
	return components.Sidebar(items)
}

func safeRoute(route string) string {
	parsed, err := url.Parse(strings.TrimSpace(route))
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "/admin"
	}
	return parsed.RequestURI()
}

// RenderModuleSidebar is a writer-friendly bridge for existing HTML shells.
func RenderModuleSidebar(w io.Writer, registry *modules.Registry, activeRoute string) error {
	return ModuleSidebar(registry, activeRoute).Render(context.Background(), w)
}
