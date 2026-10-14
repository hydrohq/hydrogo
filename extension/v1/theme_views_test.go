package v1

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeViewTheme(t *testing.T, manifest string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	themeRoot := filepath.Join(root, "viewtheme")
	files["theme.json"] = manifest
	files["layouts/base.html"] = `<main>{{slot "content"}}</main>`
	for name, content := range files {
		path := filepath.Join(themeRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestThemeManifestRejectsInvalidViews(t *testing.T) {
	for _, views := range [][]string{{"../news-list"}, {"News"}, {"news-list", "news-list"}, {"layouts/x"}} {
		manifest := ThemeManifest{Name: "custom", Version: "1.0.0", Views: views}
		if err := manifest.Validate(); err == nil {
			t.Errorf("views %q accepted", views)
		}
	}
	tooMany := make([]string, maxThemeViews+1)
	for i := range tooMany {
		tooMany[i] = "view-" + strconv.Itoa(i)
	}
	if err := (ThemeManifest{Name: "custom", Version: "1.0.0", Views: tooMany}).Validate(); err == nil {
		t.Fatal("unbounded view list accepted")
	}
}

func TestLoadThemeParsesDeclaredViewsWithPartials(t *testing.T) {
	root := writeViewTheme(t, `{"name":"viewtheme","version":"1.0.0","views":["news-list"]}`, map[string]string{
		"layouts/news-list.html":       `{{template "chrome" .Site}}{{range .Items}}<h2>{{.Title}}</h2>{{end}}{{.Site.Widgets}}`,
		"layouts/partials/chrome.html": `{{define "chrome"}}<title>{{.Title}}</title>{{end}}`,
		"layouts/news-detail.html":     `not declared`,
		"layouts/partials/unused.html": `{{define "unused"}}x{{end}}`,
	})
	bundle, err := LoadTheme(root, "viewtheme")
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.HasView(ViewNewsList) || bundle.HasView(ViewNewsDetail) {
		t.Fatalf("declared views not honoured: list=%t detail=%t", bundle.HasView(ViewNewsList), bundle.HasView(ViewNewsDetail))
	}
	var out bytes.Buffer
	data := NewsListView{Site: ThemeSiteView{Title: "<News>", Widgets: "<aside>ok</aside>"}, Items: []NewsCardView{{Title: `<script>alert(1)</script>`}}}
	if err := bundle.RenderView(&out, ViewNewsList, data); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") || !strings.Contains(got, "&lt;News&gt;") {
		t.Fatalf("view did not escape text fields: %s", got)
	}
	if !strings.Contains(got, "<aside>ok</aside>") {
		t.Fatalf("view dropped trusted widget markup: %s", got)
	}
	if err := bundle.RenderView(&out, ViewNewsDetail, nil); !errors.Is(err, ErrThemeViewUnavailable) {
		t.Fatalf("undeclared view error = %v", err)
	}
}

func TestLoadThemeRejectsMissingOrBrokenViews(t *testing.T) {
	missing := writeViewTheme(t, `{"name":"viewtheme","version":"1.0.0","views":["news-list"]}`, map[string]string{})
	if _, err := LoadTheme(missing, "viewtheme"); err == nil {
		t.Fatal("theme with a missing declared view loaded")
	}
	broken := writeViewTheme(t, `{"name":"viewtheme","version":"1.0.0","views":["news-list"]}`, map[string]string{"layouts/news-list.html": `{{if}}`})
	if _, err := LoadTheme(broken, "viewtheme"); err == nil {
		t.Fatal("theme with an unparsable view loaded")
	}
}

func TestRenderThemeViewFallsBackToBuiltinViews(t *testing.T) {
	for _, view := range []string{ViewNewsList, ViewNewsDetail, ViewMemberAuth, ViewMemberAccount} {
		var out bytes.Buffer
		var data any
		switch view {
		case ViewNewsList:
			data = NewsListView{Site: ThemeSiteView{Title: "News", Stylesheets: []string{"/designsystem/static/designsystem.css"}}}
		case ViewNewsDetail:
			data = NewsDetailView{Article: NewsArticleView{NewsCardView: NewsCardView{Title: "Story"}, Body: "<p>Body</p>"}}
		case ViewMemberAuth:
			data = MemberAuthView{Action: "/member/login", CSRF: "token"}
		case ViewMemberAccount:
			data = MemberAccountView{Email: "reader@example.test", CSRF: "token"}
		}
		if err := RenderThemeView(nil, &out, view, data); err != nil {
			t.Fatalf("builtin %s: %v", view, err)
		}
		if !strings.Contains(out.String(), "<!doctype html>") {
			t.Fatalf("builtin %s rendered no document: %s", view, out.String())
		}
	}
	var out bytes.Buffer
	if err := RenderThemeView(nil, &out, "unknown", nil); !errors.Is(err, ErrThemeViewUnavailable) {
		t.Fatalf("unknown builtin view error = %v", err)
	}
}

func TestShippedThemesProvideEveryPageView(t *testing.T) {
	for _, name := range []string{"corporate", "newsroom"} {
		bundle, err := LoadTheme("../../themes", name)
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		for _, view := range []string{ViewNewsList, ViewNewsDetail, ViewMemberAuth, ViewMemberAccount} {
			if !bundle.HasView(view) {
				t.Errorf("theme %s is missing view %s", name, view)
			}
		}
	}
}
