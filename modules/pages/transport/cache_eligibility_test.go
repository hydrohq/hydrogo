package transport

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

type cacheEligibilityPagesStub struct{ service.ContentRepository }

func (cacheEligibilityPagesStub) GetPublishedByRoute(_ context.Context, _ database.Executor, locale, slug string) (pages.Page, error) {
	return pages.Page{ID: "page-1", Locale: locale, Slug: slug, Title: "About", Body: "<p>Hello</p>", Status: pages.StatusPublished, PublishedRevision: intPtr(1)}, nil
}

func intPtr(v int) *int { return &v }

// TestPublishedPageHTMLOmitsInlineNonceForAnonymousRequest proves an
// anonymous, no-consent-banner render of a published page (the shape
// internal/server/static_cache.go treats as a disk-cache candidate per
// staticCacheURLPath) never embeds an inline nonce'd tag. Its only would-be
// source, the always-present reading-tools CSS/JS, must be referenced as
// external /designsystem/static/article-reading.{css,js} files instead.
func TestPublishedPageHTMLOmitsInlineNonceForAnonymousRequest(t *testing.T) {
	h := Handler{
		Content: service.ContentService{Pages: cacheEligibilityPagesStub{}},
	}
	r := httptest.NewRequest("GET", "/public/pages/en/about/html", nil)
	r.SetPathValue("locale", "en")
	r.SetPathValue("slug", "about")
	w := httptest.NewRecorder()
	h.getPublishedHTML(w, r)

	if w.Code != 0 && w.Code != 200 {
		t.Fatalf("expected a 200 response, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`href="/designsystem/static/article-reading.css"`, `src="/designsystem/static/article-reading.js"`} {
		if !strings.Contains(body, want) {
			t.Errorf("published page response missing %q", want)
		}
	}
	if strings.Contains(body, "nonce=") {
		t.Error("anonymous page render must not embed an inline nonce'd tag, it must stay cache-eligible")
	}
}
