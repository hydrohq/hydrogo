package ui

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

// adminAsset returns an embedded admin asset, e.g. "js/studio.js" or
// "shell.js", as served under /admin-assets/.
func adminAsset(t *testing.T, name string) string {
	t.Helper()
	data, err := fs.ReadFile(staticassets.FS, "admin/"+name)
	if err != nil {
		t.Fatalf("read admin asset %s: %v", name, err)
	}
	return string(data)
}

// inlineAssets matches inline <script> and <style> blocks so tests can look at
// the visible markup only.
var inlineAssets = regexp.MustCompile(`(?is)<script(?:\s[^>]*)?>.*?</script>|<style(?:\s[^>]*)?>.*?</style>`)

func TestAdminShellLoadsEveryShellScriptDeferredAndInOrder(t *testing.T) {
	var out strings.Builder
	if err := AdminStart(&out, "Scripts", "", ""); err != nil {
		t.Fatal(err)
	}
	markup := out.String()
	last := -1
	for _, src := range []string{
		"/vendor/htmx/htmx.min.js", "/admin-assets/motion.js", "/admin-assets/js/block-editor.js", "/admin-assets/js/icons.js",
		"/admin-assets/js/studio.js", "/admin-assets/js/ux.js", "/admin-assets/js/session-warning.js",
		"/admin-assets/shell.js", "/vendor/alpinejs/cdn.min.js",
	} {
		tag := `<script defer src="` + src + `"></script>`
		at := strings.Index(markup, tag)
		if at < 0 {
			t.Fatalf("admin shell does not load %s deferred", src)
		}
		if at < last {
			t.Fatalf("admin shell loads %s out of order", src)
		}
		last = at
		if strings.HasPrefix(src, "/admin-assets/") {
			adminAsset(t, strings.TrimPrefix(src, "/admin-assets/"))
		}
	}
	if strings.Contains(markup, "english-ui.js") {
		t.Fatal("Turkish admin shell loads the English copy translator")
	}
	scripts := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindAllStringSubmatch(markup, -1)
	if len(scripts) != 1 || !strings.Contains(scripts[0][1], "data-theme") {
		t.Fatalf("admin shell must keep exactly one inline script, the theme bootstrap; found %d", len(scripts))
	}
}

func TestAdminShellLoadsEnglishTranslatorOnlyForEnglish(t *testing.T) {
	var out strings.Builder
	if err := AdminStartLocale(&out, "Scripts", "", "", "en"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<script defer src="/admin-assets/js/english-ui.js"></script>`) {
		t.Fatal("English admin shell does not load the copy translator")
	}
	if !strings.Contains(adminAsset(t, "js/english-ui.js"), "MutationObserver") {
		t.Fatal("English translator no longer follows content swapped in by the router")
	}
}

func TestAdminShellAlpineComponentsAreRegisteredForTheCSPBuild(t *testing.T) {
	var out strings.Builder
	if err := AdminStart(&out, "Alpine", "", ""); err != nil {
		t.Fatal(err)
	}
	shellJS := adminAsset(t, "shell.js")
	for _, component := range regexp.MustCompile(`x-data="([^"]+)"`).FindAllStringSubmatch(out.String(), -1) {
		if !strings.Contains(shellJS, `Alpine.data("`+component[1]+`"`) {
			t.Errorf("x-data=%q has no Alpine.data registration in shell.js", component[1])
		}
	}
	// The CSP build cannot evaluate arbitrary JavaScript: directive values must
	// be plain property or method names.
	for _, directive := range regexp.MustCompile(`x-(?:on:[a-z.]+|bind:[a-z-]+|text|show|data)="([^"]+)"`).FindAllStringSubmatch(out.String(), -1) {
		if !regexp.MustCompile(`^[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*$`).MatchString(directive[1]) {
			t.Errorf("Alpine directive %q is an expression, not a property or method name", directive[0])
		}
	}
	if !strings.Contains(shellJS, `document.addEventListener("alpine:init"`) {
		t.Fatal("shell.js must register its components on alpine:init")
	}
}
