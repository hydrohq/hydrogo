package v1

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"sort"
	"sync"
)

// Page views a theme can provide for module-rendered public pages. A theme
// declares the ones it ships in theme.json ("views") and stores each as
// layouts/<view>.html. Shared {{define}} blocks live in layouts/partials/*.html.
const (
	ViewNewsList      = "news-list"
	ViewNewsDetail    = "news-detail"
	ViewMemberAuth    = "member-auth"
	ViewMemberAccount = "member-account"
)

const (
	maxThemeViews        = 32
	maxThemeViewBytes    = 512 << 10
	maxThemeViewPartials = 32
)

var ErrThemeViewUnavailable = errors.New("theme view template unavailable")

//go:embed builtin/layouts/*.html builtin/layouts/partials/*.html
var builtinViewFS embed.FS

var builtinViews = sync.OnceValues(func() (map[string]*template.Template, error) {
	sub, err := fs.Sub(builtinViewFS, "builtin")
	if err != nil {
		return nil, err
	}
	return parseThemeViews(sub, []string{ViewNewsList, ViewNewsDetail, ViewMemberAuth, ViewMemberAccount})
})

func themeViewFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
}

// parseThemeViews parses every declared view together with the theme's
// partials. Paths are fixed patterns below layouts/, so a manifest cannot
// point the loader at files outside the theme.
func parseThemeViews(fsys fs.FS, views []string) (map[string]*template.Template, error) {
	if len(views) == 0 {
		return nil, nil
	}
	partials, err := fs.Glob(fsys, "layouts/partials/*.html")
	if err != nil {
		return nil, err
	}
	if len(partials) > maxThemeViewPartials {
		return nil, fmt.Errorf("%w: too many view partials", ErrInvalidThemeManifest)
	}
	sort.Strings(partials)
	partialSources := make(map[string]string, len(partials))
	for _, name := range partials {
		source, err := readThemeViewFile(fsys, name)
		if err != nil {
			return nil, err
		}
		partialSources[name] = source
	}
	parsed := make(map[string]*template.Template, len(views))
	for _, view := range views {
		if !themeName.MatchString(view) {
			return nil, ErrInvalidThemeManifest
		}
		source, err := readThemeViewFile(fsys, path.Join("layouts", view+".html"))
		if err != nil {
			return nil, err
		}
		tpl := template.New(view).Funcs(themeViewFuncs())
		for _, name := range partials {
			if _, err := tpl.New(name).Parse(partialSources[name]); err != nil {
				return nil, fmt.Errorf("parse theme partial %q: %w", name, err)
			}
		}
		if _, err := tpl.Parse(source); err != nil {
			return nil, fmt.Errorf("parse theme view %q: %w", view, err)
		}
		parsed[view] = tpl
	}
	return parsed, nil
}

func readThemeViewFile(fsys fs.FS, name string) (string, error) {
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return "", fmt.Errorf("theme view %q: %w", name, err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxThemeViewBytes {
		return "", fmt.Errorf("%w: theme view %q", ErrInvalidThemeManifest, name)
	}
	content, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", fmt.Errorf("theme view %q: %w", name, err)
	}
	return string(content), nil
}

// HasView reports whether the theme ships the named page view.
func (b ThemeBundle) HasView(name string) bool {
	_, ok := b.views[name]
	return ok
}

// RenderView executes a declared page view with a typed view model. Output
// is buffered so a template error never leaves a half-written page.
func (b ThemeBundle) RenderView(w io.Writer, name string, data any) error {
	tpl, ok := b.views[name]
	if !ok || w == nil {
		return ErrThemeViewUnavailable
	}
	return executeThemeView(w, tpl, data)
}

// RenderThemeView renders a page view from the active theme, falling back to
// the minimal built-in view when no theme is loaded or the theme does not
// ship that view.
func RenderThemeView(theme *ThemeBundle, w io.Writer, name string, data any) error {
	if theme != nil && theme.HasView(name) {
		return theme.RenderView(w, name, data)
	}
	views, err := builtinViews()
	if err != nil {
		return err
	}
	tpl, ok := views[name]
	if !ok || w == nil {
		return ErrThemeViewUnavailable
	}
	return executeThemeView(w, tpl, data)
}

func executeThemeView(w io.Writer, tpl *template.Template, data any) error {
	var output bytes.Buffer
	if err := tpl.Execute(&output, data); err != nil {
		return err
	}
	_, err := w.Write(output.Bytes())
	return err
}
