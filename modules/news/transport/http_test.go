package transport

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type publicMediaURLStub struct {
	URL string
	Err error
}

func TestFloatingContactUsesConfiguredSupportOnly(t *testing.T) {
	handler := Handler{PublicSupport: adminstudio.SupportConfig{Enabled: true, Phone: "+1 415 555 0132", WhatsAppURL: "https://wa.me/14155550132"}}
	got := handler.floatingContact(context.Background(), "en")
	if !strings.Contains(got, `aria-label="Contact options"`) || !strings.Contains(got, `href="tel:+14155550132"`) {
		t.Fatalf("configured public contact missing: %s", got)
	}
	if got := (Handler{}).floatingContact(context.Background(), "tr"); got != "" {
		t.Fatalf("disabled support rendered: %s", got)
	}
}

func (s publicMediaURLStub) PublicURL(string) (string, error) { return s.URL, s.Err }

func TestNewsCoverMarkupUsesConfiguredCDNAndEscapesMarkup(t *testing.T) {
	h := Handler{Media: publicMediaURLStub{URL: "https://cdn.example.test/media/hero.webp"}}
	got := string(h.newsCoverMarkup(news.Item{CoverMediaID: "hero", Title: "<headline>"}))
	if !strings.Contains(got, `src="https://cdn.example.test/media/hero.webp"`) {
		t.Fatalf("cdn URL missing: %s", got)
	}
	if strings.Contains(got, "<headline>") || !strings.Contains(got, "&lt;headline&gt;") {
		t.Fatalf("title was not escaped: %s", got)
	}
}

func TestNewsCoverMarkupFallsBackWhenCDNUnavailable(t *testing.T) {
	h := Handler{Media: publicMediaURLStub{Err: errors.New("unavailable")}}
	got := string(h.newsCoverMarkup(news.Item{CoverMediaID: "hero", Title: "Headline"}))
	if !strings.Contains(got, `src="/media/hero?variant=card"`) {
		t.Fatalf("local fallback missing: %s", got)
	}
}

func TestReadingMinutesUsesAtLeastOneMinute(t *testing.T) {
	if got := readingMinutes(""); got != 1 {
		t.Fatalf("empty excerpt reading time = %d, want 1", got)
	}
	words := strings.Repeat("word ", 201)
	if got := readingMinutes(words); got != 2 {
		t.Fatalf("201-word excerpt reading time = %d, want 2", got)
	}
}

type taxonomyTransactionProbe struct{}

func (taxonomyTransactionProbe) ListTaxonomy(context.Context, database.Executor, string) ([]v1.TaxonomyTerm, error) {
	return nil, nil
}

func (taxonomyTransactionProbe) ListTaxonomyTerms(context.Context, database.Executor) ([]v1.TaxonomyTerm, error) {
	return nil, nil
}

func (taxonomyTransactionProbe) ReplaceTaxonomy(ctx context.Context, executor database.Executor, _ string, _ []v1.TaxonomyTerm) error {
	if _, err := executor.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS taxonomy_transaction_probe (id INTEGER PRIMARY KEY)`); err != nil {
		return err
	}
	_, err := executor.ExecContext(ctx, `INSERT INTO taxonomy_transaction_probe (id) VALUES (1)`)
	return err
}

func (taxonomyTransactionProbe) UpsertTaxonomyTerm(context.Context, database.Executor, v1.TaxonomyTerm) error {
	return nil
}

func (taxonomyTransactionProbe) DeleteTaxonomyTerm(context.Context, database.Executor, string) error {
	return nil
}

func TestReplaceTaxonomyAtomicallyUsesDatabaseTransaction(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "taxonomy-transaction.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := replaceTaxonomyAtomically(ctx, db, taxonomyTransactionProbe{}, "article-1", nil); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM taxonomy_transaction_probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("transaction probe count = %d, want 1", count)
	}
}

func TestRoutesIncludeCSRFProtectedAdminNewsForms(t *testing.T) {
	routes := (Handler{}).Routes()
	want := map[string]bool{
		"GET /admin/news/{id}":                  true,
		"POST /admin/news/{id}":                 true,
		"GET /admin/news/taxonomy":              true,
		"POST /admin/news/taxonomy":             true,
		"POST /admin/news/taxonomy/{id}/delete": true,
		"POST /admin/news/{id}/taxonomy":        true,
		"POST /admin/news/{id}/schedule":        true,
		"POST /admin/news/{id}/publish":         true,
		"POST /admin/news/{id}/retract":         true,
	}
	for _, route := range routes {
		if _, ok := want[route.Pattern]; ok {
			if !route.Protected {
				t.Errorf("route %q is not protected", route.Pattern)
			}
			delete(want, route.Pattern)
		}
	}
	for pattern := range want {
		t.Errorf("missing admin route %q", pattern)
	}
}

func TestWorkflowActionMarkupFollowsConfiguredPolicy(t *testing.T) {
	service := news.Service{Workflow: v1.EditorialWorkflow{
		States:      []v1.EditorialState{v1.EditorialDraft, v1.EditorialPublished},
		Transitions: map[v1.EditorialState][]v1.EditorialState{v1.EditorialDraft: {}},
	}}
	markup := string(workflowActionMarkup(service, news.Item{ID: "article-1", Status: news.StatusDraft}, "csrf"))
	if strings.Contains(markup, "/publish") || strings.Contains(markup, "/schedule") || strings.Contains(markup, "/retract") {
		t.Fatalf("disallowed workflow action rendered: %s", markup)
	}
	if !strings.Contains(markup, "kullanılabilir yayın aksiyonu yok") {
		t.Fatalf("missing empty workflow action state: %s", markup)
	}
}

func TestRoutesIncludePublicPublishedNewsList(t *testing.T) {
	foundList, foundBySlug := false, false
	for _, route := range (Handler{}).Routes() {
		if route.Pattern == "GET /api/news" {
			if route.Protected {
				t.Fatal("published news list must be public")
			}
			foundList = true
		}
		if route.Pattern == "GET /api/news/by-slug" {
			if route.Protected {
				t.Fatal("published news lookup must be public")
			}
			foundBySlug = true
		}
	}
	if !foundList || !foundBySlug {
		t.Fatalf("missing public news routes list=%t by_slug=%t", foundList, foundBySlug)
	}
}

func TestSortNewsByViewsHonorsDirectionAndStableTies(t *testing.T) {
	items := []news.Item{{ID: "c"}, {ID: "a"}, {ID: "b"}}
	counts := map[string]int64{"a": 5, "b": 5, "c": 2}
	sortNewsByViews(items, counts, "-views")
	if items[0].ID != "a" || items[1].ID != "b" || items[2].ID != "c" {
		t.Fatalf("descending view sort = %#v", items)
	}
	sortNewsByViews(items, counts, "views")
	if items[0].ID != "c" || items[1].ID != "a" || items[2].ID != "b" {
		t.Fatalf("ascending view sort = %#v", items)
	}
}
