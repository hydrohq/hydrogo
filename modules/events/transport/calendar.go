package transport

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/events"
)

type PublishedEvents interface {
	ListPublished(ctx context.Context, db database.QueryExecutor, locale string, limit int) ([]events.Event, error)
}

type CalendarHandler struct {
	DB     database.QueryExecutor
	Events PublishedEvents
	Locale string
}

func (h CalendarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.DB == nil || h.Events == nil {
		http.Error(w, "Calendar unavailable", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimSuffix(r.URL.Path, ".ics")
	slug := strings.Trim(strings.TrimPrefix(path, "/events/"), "/")
	canonicalSlug, err := canonical.Path(slug)
	if err != nil || canonicalSlug == "" || canonicalSlug != slug || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}
	items, err := h.Events.ListPublished(r.Context(), h.DB, h.Locale, 200)
	if err != nil {
		http.Error(w, "Calendar unavailable", http.StatusServiceUnavailable)
		return
	}
	for _, item := range items {
		if item.Slug != slug {
			continue
		}
		body := events.CalendarInvite(item)
		if body == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="event.ics"`)
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write([]byte(body))
		return
	}
	http.NotFound(w, r)
}

func CalendarURL(slug string) string { return "/events/" + url.PathEscape(slug) + ".ics" }
