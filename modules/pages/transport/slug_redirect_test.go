package transport

import (
	"context"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestPublishedDefaultLocaleSlugChangeCreatesRedirect(t *testing.T) {
	calls := 0
	var gotLocale, gotOld, gotNew string
	h := Handler{DefaultLocale: "en", RecordSlugRedirect: func(_ context.Context, locale, oldSlug, newSlug string) error {
		calls++
		gotLocale, gotOld, gotNew = locale, oldSlug, newSlug
		return nil
	}}
	previous := pages.Page{ID: "p1", Locale: "en", Slug: "old-title", Status: pages.StatusPublished}
	if err := h.recordPublishedSlugChange(context.Background(), previous, "en", "new-title"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || gotLocale != "en" || gotOld != "old-title" || gotNew != "new-title" {
		t.Fatalf("calls=%d locale=%q old=%q new=%q", calls, gotLocale, gotOld, gotNew)
	}
	previous.Status = pages.StatusDraft
	if err := h.recordPublishedSlugChange(context.Background(), previous, "en", "another"); err != nil {
		t.Fatal(err)
	}
	previous.Status = pages.StatusPublished
	previous.Locale = "fr"
	if err := h.recordPublishedSlugChange(context.Background(), previous, "fr", "localized"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("non-published/localized changes recorded redirects: %d", calls)
	}
}
