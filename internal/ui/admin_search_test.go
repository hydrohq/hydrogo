package ui

import (
	"strings"
	"testing"
)

func TestSpotlightSearchUsesAccessibleDialogAndGlobalShortcuts(t *testing.T) {
	for _, marker := range []string{"document.createElement('dialog')", "document.documentElement.lang === 'en' ? 'Search Studio' : 'Studio içinde ara'", "Quick actions & commands", "Search screens and commands (⌘K)", "toLocaleLowerCase(searchLocale)", "No matching results.", "showModal()", "spotlightDialog.close()", "e.key.toLowerCase() === 'k'", "e.key === '/'", "role', 'listbox'"} {
		if !strings.Contains(AdminSPARouterJS, marker) {
			t.Errorf("spotlight script omitted %q", marker)
		}
	}
	for _, marker := range []string{".studio-search-dialog::backdrop", ".studio-search-dialog .studio-search-results", "prefers-reduced-motion:reduce"} {
		if !strings.Contains(AdminSearchCSS, marker) {
			t.Errorf("spotlight styles omitted %q", marker)
		}
	}
}
