package v1

import (
	"encoding/json"
	"errors"
	htmltemplate "html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

var ErrThemeBlockUnavailable = errors.New("theme block template unavailable")

// RenderPage renders a loaded theme layout with named, already-sanitized
// slots. Slot names are validated by the manifest and unknown slots resolve
// to empty output.
func (b ThemeBundle) RenderPage(w io.Writer, data PageTemplateData) error {
	if w == nil || data.Slots == nil {
		return ErrInvalidThemeManifest
	}
	if data.Direction == "" {
		data.Direction = "ltr"
	}
	if data.Direction != "ltr" && data.Direction != "rtl" {
		return ErrInvalidThemeManifest
	}
	tpl, err := htmltemplate.New(b.Manifest.Name).Funcs(htmltemplate.FuncMap{
		"slot": func(name string) htmltemplate.HTML { return data.Slots[name] },
	}).Parse(b.Layout)
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}

// ServeAsset exposes only files declared in theme.json. The route is expected
// to pass the path below /theme/static, so a theme asset cannot escape its
// bundle or become an arbitrary filesystem read.
func (b ThemeBundle) ServeAsset(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.PathValue("path"), "/")
	if rel == "" || !safeThemePath(rel) {
		http.NotFound(w, r)
		return
	}
	asset := filepath.ToSlash(filepath.Join("static", rel))
	declared := false
	for _, candidate := range b.Manifest.Assets {
		if filepath.ToSlash(filepath.Clean(candidate)) == asset {
			declared = true
			break
		}
	}
	if !declared {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(b.Root, filepath.FromSlash(asset))
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
}

type BlockTemplateData struct {
	Type    string
	Payload map[string]any
}

// RenderBlock renders a theme-provided block template. Missing templates are
// reported distinctly so callers can safely fall back to the core renderer.
func (b ThemeBundle) RenderBlock(w io.Writer, block pages.Block) error {
	if w == nil || pages.ValidateBlocks([]pages.Block{block}) != nil {
		return ErrThemeBlockUnavailable
	}
	path := filepath.Join(b.Root, "blocks", block.Type+".html")
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrThemeBlockUnavailable
	}
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(block.Payload, &payload); err != nil {
		return ErrThemeBlockUnavailable
	}
	tpl, err := htmltemplate.New(block.Type).Parse(string(content))
	if err != nil {
		return err
	}
	return tpl.Execute(w, BlockTemplateData{Type: block.Type, Payload: payload})
}
