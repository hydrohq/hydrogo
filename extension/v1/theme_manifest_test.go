package v1

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeManifestValidation(t *testing.T) {
	manifest := ThemeManifest{Name: "corporate", Version: "1.0.0", Tokens: map[string]string{"primary": "#123456"}, Slots: []string{"header", "content"}, Assets: []string{"theme.css"}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	manifest.Name = "../unsafe"
	if err := manifest.Validate(); err == nil {
		t.Fatal("unsafe manifest accepted")
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
	if err := os.WriteFile(filepath.Join(themeRoot, "layouts", "base.html"), []byte(`{{slot "header"}}<main>{{slot "content"}}</main>`), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadTheme(root, "corporate")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := bundle.RenderPage(&out, PageTemplateData{Slots: map[string]template.HTML{"header": "<header>H</header>", "content": "<p>C</p>"}}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "<header>H</header><main><p>C</p></main>" {
		t.Fatalf("rendered theme = %q", out.String())
	}
}

func TestThemeManifestRejectsTraversalAsset(t *testing.T) {
	manifest := ThemeManifest{Name: "corporate", Version: "1.0.0", Assets: []string{"../secret.css"}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("traversal asset accepted")
	}
}
