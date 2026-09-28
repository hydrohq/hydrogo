package transport

import (
	"context"
	"database/sql"
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
	handler := CalendarHandler{DB: &sql.DB{}, Events: eventSource{item}, Locale: "en"}
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
