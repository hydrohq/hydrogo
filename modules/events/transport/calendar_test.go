package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/modules/events"
)

type eventSource []events.Event

func (e eventSource) ListPublished(context.Context, database.QueryExecutor, string, int) ([]events.Event, error) {
	return e, nil
}

func TestCalendarHandlerExportsOnlyPublishedSlug(t *testing.T) {
	item := events.Event{Base: sectoral.Base{ID: "event-1", Locale: "en", Slug: "talk", Title: "Public event", Status: "published"}, StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour), Timezone: "UTC"}
	calendar := CalendarHandler{DB: &sql.DB{}, Events: eventSource{item}, Locale: "en"}
	ext := events.Extension{Calendar: calendar}
	routes := ext.Routes()
	manifest := ext.ModuleManifest()
	if len(routes) != 1 || len(manifest.Routes) != 1 || routes[0].Pattern != manifest.Routes[0].Pattern {
		t.Fatal("calendar runtime and manifest routes must agree")
	}
	handler := http.NewServeMux()
	handler.Handle(routes[0].Pattern, routes[0].Handler)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/events/talk.ics", nil))
	if rr.Code != 200 || rr.Header().Get("Content-Type") != "text/calendar; charset=utf-8" || !strings.Contains(rr.Body.String(), "SUMMARY:Public event") {
		t.Fatalf("response=%d %s %s", rr.Code, rr.Header(), rr.Body.String())
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/events/missing.ics", nil))
	if rr.Code != 404 {
		t.Fatalf("missing event status=%d", rr.Code)
	}
}
