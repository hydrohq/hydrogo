package transport

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type publicNewsRepo struct {
	news.Repository
	items []news.Item
}

func (r publicNewsRepo) ListPublished(context.Context, database.Executor, string, int) ([]news.Item, error) {
	return r.items, nil
}

func (r publicNewsRepo) ListRelatedPublished(context.Context, database.Executor, string, string, int) ([]news.Item, error) {
	return r.items, nil
}

func (r publicNewsRepo) GetPublishedByRoute(_ context.Context, _ database.Executor, locale, slug string) (news.Item, error) {
	for _, item := range r.items {
		if item.Slug == slug {
			item.Locale = locale
			return item, nil
		}
	}
	return news.Item{}, errors.New("not found")
}

func loadShippedTheme(t *testing.T, name string) *v1.ThemeBundle {
	t.Helper()
	bundle, err := v1.LoadTheme(filepath.Join("..", "..", "..", "themes"), name)
	if err != nil {
		t.Fatal(err)
	}
	return &bundle
}

func publicNewsHandler(t *testing.T, theme *v1.ThemeBundle) Handler {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "public-news.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	items := []news.Item{
		{ID: "a", Slug: "first-story", Title: "First <story>", Body: "Body text", Category: "Ekonomi", AuthorName: "Ayşe", Tags: "go, cms", KeyTakeaways: "- One\n- Two", Featured: true},
		{ID: "b", Slug: "second-story", Title: "Second story", Body: "More text", Category: "Dünya Haberleri", CoverMediaID: "cover-1"},
	}
	return Handler{DB: db, Service: news.Service{Repository: publicNewsRepo{items: items}}, Theme: theme}
}

var inlineScriptBody = regexp.MustCompile(`(?s)<script(?:\s[^>]*)?>\s*[^<\s]`)

func renderPublicNews(t *testing.T, h Handler, path, locale, slug string) string {
	t.Helper()
	request := httptest.NewRequest("GET", path, nil)
	request.SetPathValue("locale", locale)
	response := httptest.NewRecorder()
	if slug == "" {
		h.publicListHTML(response, request)
	} else {
		request.SetPathValue("slug", slug)
		h.publicGetHTML(response, request)
	}
	if response.Code != 200 {
		t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
	}
	body := response.Body.String()
	if inlineScriptBody.MatchString(body) || strings.Contains(body, "onclick=") || strings.Contains(body, "onerror=") || strings.Contains(body, "<style") {
		t.Fatalf("%s renders inline code that the strict CSP would block", path)
	}
	return body
}

func TestPublicNewsPagesRenderThroughEveryTheme(t *testing.T) {
	for _, theme := range []string{"", "newsroom"} {
		var bundle *v1.ThemeBundle
		if theme != "" {
			bundle = loadShippedTheme(t, theme)
		}
		h := publicNewsHandler(t, bundle)
		list := renderPublicNews(t, h, "/public/news/tr/html", "tr", "")
		for _, want := range []string{`data-testid="public-news"`, `href="/public/news/tr/first-story/html"`, "First &lt;story&gt;", `src="/media/cover-1?variant=card"`} {
			if !strings.Contains(list, want) {
				t.Errorf("theme %q list missing %q", theme, want)
			}
		}
		detail := renderPublicNews(t, h, "/public/news/tr/first-story/html", "tr", "first-story")
		for _, want := range []string{"First &lt;story&gt;", `href="/designsystem/static/article-reading.css"`, `src="/designsystem/static/article-reading.js"`, "#go", "One", `href="/public/news/tr/second-story/html"`} {
			if !strings.Contains(detail, want) {
				t.Errorf("theme %q detail missing %q", theme, want)
			}
		}
		if strings.Contains(list+detail, "<story>") {
			t.Errorf("theme %q rendered an unescaped title", theme)
		}
	}
}

func TestNewsLayoutRendersLocalizedNestedNavigation(t *testing.T) {
	for _, theme := range []string{"newsroom"} {
		h := publicNewsHandler(t, loadShippedTheme(t, theme))
		h.NavigationProvider = func(context.Context, string) ([]v1.ThemeNavigationItem, error) {
			return []v1.ThemeNavigationItem{
				{ID: "root", Label: "Services", Href: "/en/services", Position: 0},
				{ID: "child", ParentID: "root", Label: "Consulting", Href: "/en/consulting", Position: 1, Target: "_blank"},
			}, nil
		}
		page := renderPublicNews(t, h, "/public/news/en/html", "en", "")
		for _, marker := range []string{`lang="en"`, `href="/en/services"`, `class="site-navigation-child-list"`, `rel="noopener noreferrer"`, "Consulting"} {
			if !strings.Contains(page, marker) {
				t.Errorf("theme %s news layout missing %q", theme, marker)
			}
		}
		if strings.Contains(page, "/public/pages/tr/") || strings.Contains(page, "/public/news/tr/") {
			t.Errorf("theme %s kept Turkish routes on an English page", theme)
		}
	}
}

func TestNewsroomFallsBackToCategoriesFromContent(t *testing.T) {
	h := publicNewsHandler(t, loadShippedTheme(t, "newsroom"))
	page := renderPublicNews(t, h, "/public/news/tr/html?category=ekonomi", "tr", "")
	for _, marker := range []string{`href="/public/news/tr/html?category=ekonomi"`, `href="/public/news/tr/html?category=dunya-haberleri"`, "Dünya Haberleri"} {
		if !strings.Contains(page, marker) {
			t.Errorf("newsroom category navigation missing %q", marker)
		}
	}
	for _, hardcoded := range []string{"GÜNDEM", "TEKNOLOJİ", "category=spor"} {
		if strings.Contains(page, hardcoded) {
			t.Errorf("newsroom still renders hardcoded section %q", hardcoded)
		}
	}
}

func TestCategoryKeyFoldsTurkishLabels(t *testing.T) {
	for label, want := range map[string]string{"Ekonomi": "ekonomi", "Dünya Haberleri": "dunya-haberleri", "İÇ POLİTİKA": "ic-politika", " Güvenlik & Mimari ": "guvenlik-mimari", "": ""} {
		if got := categoryKey(label); got != want {
			t.Errorf("categoryKey(%q) = %q, want %q", label, got, want)
		}
	}
}
