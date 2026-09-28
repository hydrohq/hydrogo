package announcements

import (
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
)

func TestVisibleAtHonorsPublicationWindow(t *testing.T) {
	now := time.Now()
	item := Announcement{Base: sectoral.Base{Status: "published"}, StartsAt: now.Add(time.Hour), EndsAt: now.Add(2 * time.Hour)}
	if item.VisibleAt(now) {
		t.Fatal("future announcement was visible")
	}
	if !item.VisibleAt(now.Add(90 * time.Minute)) {
		t.Fatal("active announcement hidden")
	}
	if item.VisibleAt(now.Add(3 * time.Hour)) {
		t.Fatal("expired announcement visible")
	}
	item.Status = "draft"
	if item.VisibleAt(now.Add(90 * time.Minute)) {
		t.Fatal("draft announcement visible")
	}
}
