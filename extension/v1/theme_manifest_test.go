package v1

import (
	"encoding/json"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeManifestValidation(t *testing.T) {
	manifest := ThemeManifest{Name: "corporate", Version: "1.0.0", DesignSystem: "shadcn", Tokens: map[string]string{"primary": "#123456"}, Slots: []string{"header", "content"}, Assets: []string{"theme.css"}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	manifest.Name = "../unsafe"
	if err := manifest.Validate(); err == nil {
		t.Fatal("unsafe manifest accepted")
	}
}

func TestThemeManifestAcceptsCustomDesignSystemIdentifier(t *testing.T) {
	manifest := ThemeManifest{Name: "custom", Version: "1.0.0", DesignSystem: "acme-system", Assets: []string{"theme.css"}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestThemeManifestRejectsInvalidDesignSystemIdentifier(t *testing.T) {
	manifest := ThemeManifest{Name: "custom", Version: "1.0.0", DesignSystem: "../material", Assets: []string{"theme.css"}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("invalid design system accepted")
	}
}

func TestLoadThemeAndRenderNamedSlots(t *testing.T) {
	root := t.TempDir()
	themeRoot := filepath.Join(root, "corporate")
	if err := os.MkdirAll(filepath.Join(themeRoot, "layouts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "theme.json"), []byte(`{"name":"corporate","version":"1.0.0","slots":["header","content"],"assets":["static/theme.css"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(themeRoot, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "static", "theme.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "layouts", "base.html"), []byte(`<html lang="{{.Locale}}" dir="{{.Direction}}">{{slot "header"}}<main>{{slot "content"}}</main></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadTheme(root, "corporate")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := bundle.RenderPage(&out, PageTemplateData{Direction: "rtl", Slots: map[string]template.HTML{"header": "<header>H</header>", "content": "<p>C</p>"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `dir="rtl"`) {
		t.Fatalf("theme output missing direction: %s", out.String())
	}
	if out.String() != `<html lang="" dir="rtl"><header>H</header><main><p>C</p></main></html>` {
		t.Fatalf("rendered theme = %q", out.String())
	}
}

func TestThemeManifestRejectsTraversalAsset(t *testing.T) {
	manifest := ThemeManifest{Name: "corporate", Version: "1.0.0", Assets: []string{"../secret.css"}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("traversal asset accepted")
	}
}

func TestThemeBundleRendersEscapedBlockOverride(t *testing.T) {
	root := t.TempDir()
	themeRoot := filepath.Join(root, "corporate")
	if err := os.MkdirAll(filepath.Join(themeRoot, "layouts", "blocks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "theme.json"), []byte(`{"name":"corporate","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "layouts", "base.html"), []byte(`{{slot "content"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(themeRoot, "blocks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "blocks", "hero.html"), []byte(`<h1>{{.Payload.title}}</h1>`), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadTheme(root, "corporate")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := bundle.RenderBlock(&out, pages.Block{Type: "hero", Payload: json.RawMessage(`{"title":"<unsafe>"}`)}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<unsafe>") || !strings.Contains(out.String(), "&lt;unsafe&gt;") {
		t.Fatalf("unsafe block output: %s", out.String())
	}
}

func TestThemeBundleServesDeclaredStaticAssetOnly(t *testing.T) {
	root := t.TempDir()
	themeRoot := filepath.Join(root, "corporate")
	if err := os.MkdirAll(filepath.Join(themeRoot, "layouts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(themeRoot, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "theme.json"), []byte(`{"name":"corporate","version":"1.0.0","assets":["static/theme.css"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "layouts", "base.html"), []byte(`{{slot "content"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "static", "theme.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadTheme(root, "corporate")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/theme/static/theme.css", nil)
	req.SetPathValue("path", "theme.css")
	res := httptest.NewRecorder()
	bundle.ServeAsset(res, req)
	if res.Code != http.StatusOK || res.Body.String() != "body{}" {
		t.Fatalf("asset response = %d %q", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/theme/static/secret.txt", nil)
	req.SetPathValue("path", "secret.txt")
	res = httptest.NewRecorder()
	bundle.ServeAsset(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("undeclared asset response = %d", res.Code)
	}
}
