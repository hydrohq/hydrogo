package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdminShellLoadsAccessibleStatusFilter(t *testing.T) {
	var output bytes.Buffer
	if err := AdminStartLocaleRole(&output, "Pages", "", "pages", "en", "admin"); err != nil {
		t.Fatal(err)
	}
	markup := output.String()
	if !strings.Contains(mustAdminCSS("ux.css"), ".status-pill--published") || !strings.Contains(markup, `href="/admin-assets/css/ux.css"`) {
		t.Error("admin shell omitted status filtering styles")
	}
	for _, want := range []string{"data-status-filter", "data-status-row", "aria-live", "data-status-filter-clear", "data-copy-text", "clipboard.writeText"} {
		if !strings.Contains(markup, want) {
			t.Errorf("admin shell omitted status filtering support %q", want)
		}
	}
}

func TestViewerStudioNavigationShowsContentAndHidesSystemManagement(t *testing.T) {
	var output bytes.Buffer
	if err := AdminStartLocaleRole(&output, "Articles", "", "generated", "tr", "viewer"); err != nil {
		t.Fatal(err)
	}
	if err := AdminEnd(&output); err != nil {
		t.Fatal(err)
	}
	markup := output.String()
	start := strings.Index(markup, `<aside`)
	if start < 0 {
		t.Fatal("shared shell did not render its navigation")
	}
	end := strings.Index(markup[start:], `</aside>`)
	if end < 0 {
		t.Fatal("shared shell navigation did not close")
	}
	navigation := markup[start : start+end]
	if !strings.Contains(navigation, `href="/admin/pages"`) {
		t.Fatal("Viewer cannot navigate to readable page content")
	}
	for _, href := range []string{
		"/admin/users", "/admin/permissions", "/admin/security", "/admin/audit",
		"/admin/backup", "/admin/developer", "/admin/control", "/admin/operations",
	} {
		if strings.Contains(navigation, `href="`+href+`"`) {
			t.Errorf("Viewer navigation exposes privileged route %s", href)
		}
	}
}
