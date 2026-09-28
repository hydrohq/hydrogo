package events

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
	"testing"
	"time"
)

func TestCalendarInviteEscapesTextAndUsesUTC(t *testing.T) {
	item := Event{Base: sectoral.Base{ID: "event-1", Locale: "en", Slug: "launch-day", Title: "Launch; day\r\nX-Injected: yes", Body: "A, B"}, StartsAt: time.Date(2026, 10, 1, 12, 0, 0, 0, time.FixedZone("offset", 3600)), EndsAt: time.Date(2026, 10, 1, 14, 0, 0, 0, time.FixedZone("offset", 3600)), Timezone: "UTC"}
	calendar := CalendarInvite(item)
	for _, want := range []string{"BEGIN:VCALENDAR\r\n", "DTSTART:20261001T110000Z\r\n", "SUMMARY:Launch\\; day\\nX-Injected: yes\r\n", "DESCRIPTION:\\nA\\, B\r\n"} {
		if !strings.Contains(calendar, want) {
			t.Errorf("calendar missing %q: %s", want, calendar)
		}
	}
	if strings.Contains(calendar, "\r\nX-Injected:") {
		t.Fatal("calendar text allowed line injection")
	}
}
