package search

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type SearchRecorder interface {
	Enqueue(analytics.SearchQueryEvent) bool
}

type PublicSearchHandler struct {
	Autocomplete AutocompleteHandler
	Recorder     SearchRecorder
	Now          func() time.Time
}

func (h PublicSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	term := strings.TrimSpace(r.URL.Query().Get("q"))
	if term == "" || len(term) > MaxQueryLength {
		http.Error(w, "Geçerli bir arama kelimesi girin.", http.StatusBadRequest)
		return
	}
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if locale == "" || len(locale) > 35 || strings.ContainsAny(locale, "\x00\r\n") {
		locale = "tr"
	}
	tenantID := "default"
	if h.Autocomplete.RequireTenantScope {
		scope, err := tenant.Require(r.Context())
		if err != nil {
			http.Error(w, "Arama kullanılamıyor.", http.StatusForbidden)
			return
		}
		tenantID = scope.TenantID
	} else if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	}
	items, err := h.Autocomplete.suggest(r.Context(), term, locale, tenantID)
	if err != nil {
		http.Error(w, "Arama şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	results := make([]designsystem.PublicSearchResult, 0, len(items))
	for _, item := range items {
		base := "/public/pages/"
		category := "Sayfa"
		if item.kind == "news" {
			base = "/public/news/"
			category = "Haber"
		}
		results = append(results, designsystem.PublicSearchResult{Title: item.title, Category: category, URL: base + url.PathEscape(item.locale) + "/" + url.PathEscape(item.slug) + "/html"})
	}
	if h.Recorder != nil {
		now := time.Now
		if h.Now != nil {
			now = h.Now
		}
		normalized := analytics.NormalizeSearchTerm(term)
		if normalized != "" {
			h.Recorder.Enqueue(analytics.SearchQueryEvent{Day: analytics.SearchQueryDay(now()), TenantID: tenantID, Term: normalized, Locale: locale, ResultCount: len(results)})
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := designsystem.PublicSearchResultsPage(designsystem.PublicSearchResultsProps{Term: term, Results: results}).Render(r.Context(), w); err != nil {
		http.Error(w, "Arama sonucu gösterilemiyor.", http.StatusServiceUnavailable)
	}
}
