package ui

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/site"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

// The shell links every admin stylesheet exactly once and inlines none, so
// the CSP needs no style hashes for it.
func TestAdminShellLinksEveryStylesheetWithoutInlineStyles(t *testing.T) {
	var out bytes.Buffer
	if err := AdminStart(&out, "Dashboard", "", "dashboard"); err != nil {
		t.Fatal(err)
	}
	markup := out.String()
	if strings.Contains(markup, "<style") {
		t.Fatal("admin shell renders an inline <style> block")
	}
	entries, err := fs.ReadDir(staticassets.FS, "admin/css")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "login.css" {
			continue
		}
		link := `<link rel="stylesheet" href="/admin-assets/css/` + entry.Name() + `">`
		if strings.Count(markup, link) != 1 {
			t.Errorf("admin shell must link %s exactly once", entry.Name())
		}
	}
	if !strings.Contains(markup, `href="/admin-assets/css/fonts.css"><link rel="stylesheet" href="/admin-assets/css/base.css">`) ||
		strings.Index(markup, "/admin-assets/css/base.css") > strings.Index(markup, "/admin-assets/css/avatar-contrast.css") {
		t.Error("admin stylesheets are linked out of cascade order")
	}
}

func TestAdminLoginLinksStylesheetWithoutInlineStyles(t *testing.T) {
	var out bytes.Buffer
	if err := RenderAdminLogin(&out, "csrf", "/admin"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<link rel="stylesheet" href="/admin-assets/css/login.css">`) {
		t.Fatal("login page does not link its stylesheet")
	}
	if strings.Contains(out.String(), "<style") {
		t.Fatal("login page without a background image renders an inline <style> block")
	}
	if !strings.Contains(mustAdminCSS("login.css"), ".login-hero-pane") {
		t.Fatal("login stylesheet is missing the hero pane rules")
	}
}

func TestAdminLoginBackgroundNeedsNoInlineStyleBlock(t *testing.T) {
	var out bytes.Buffer
	appearance := site.Appearance{LoginBackgroundURL: "/media/hero.webp", LoginOverlayOpacity: .35}
	if err := RenderAdminLoginWithAppearance(&out, "csrf", "/admin", appearance); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<style") || strings.Contains(AdminLoginCSP(out.Bytes()), "style-src") {
		t.Fatal("login background must not need an inline <style> block or a style hash")
	}
	if !strings.Contains(out.String(), `<aside class="login-hero-pane" style='--login-background:`) || !strings.Contains(mustAdminCSS("login.css"), "var(--login-background)") {
		t.Fatal("login background is not passed to login.css as --login-background")
	}
}
