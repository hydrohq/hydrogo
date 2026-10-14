package ui

import (
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

// The Admin Studio stylesheets live in static/admin/css and are linked by the
// shell (and the login page) from /admin-assets/css/. These variables expose
// the embedded file contents to tests that assert CSS contracts; nothing
// renders them inline.
var (
	AdminInterFontCSS  = mustAdminCSS("fonts.css")
	AdminCSS           = mustAdminCSS("base.css")
	AdminCSSOverrides  = mustAdminCSS("overrides.css")
	AdminPolishCSS     = mustAdminCSS("polish.css")
	AdminSearchCSS     = mustAdminCSS("search.css")
	AdminInfoTipCSS    = mustAdminCSS("areas.css")
	AdminHydroGoCSS    = mustAdminCSS("hydrogo-theme.css")
	AdminKitCSS        = mustAdminCSS("kit.css")
	AdminMediaFormsCSS = mustAdminCSS("media.css")
	AdminResponsiveCSS = mustAdminCSS("responsive.css")
)

func mustAdminCSS(name string) string {
	data, err := staticassets.FS.ReadFile("admin/css/" + name)
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	return string(data)
}
