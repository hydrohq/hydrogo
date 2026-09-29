package adminstudio

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWeeklyDigestWindowUsesLocalMondayNineAndCalendarWeek(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	before := time.Date(2026, 3, 9, 8, 59, 0, 0, loc)
	start, end, key, due, err := WeeklyDigestWindow(before, "America/New_York")
	if err != nil || due || key != "2026-03-02" {
		t.Fatalf("before window=%s %s %s due=%v err=%v", start, end, key, due, err)
	}
	after := time.Date(2026, 3, 9, 9, 0, 0, 0, loc)
	start, end, key, due, err = WeeklyDigestWindow(after, "America/New_York")
	if err != nil || !due || key != "2026-03-02" || start.Location().String() != loc.String() || end.Hour() != 0 {
		t.Fatalf("after window=%s %s %s due=%v err=%v", start, end, key, due, err)
	}
	if end.Sub(start) != 167*time.Hour {
		t.Fatalf("DST week duration=%v", end.Sub(start))
	}
}

func TestMonthlyDigestWindowUsesLocalFirstAtNine(t *testing.T) {
	before := time.Date(2026, 3, 1, 8, 59, 0, 0, time.UTC)
	start, end, key, due, err := MonthlyDigestWindow(before, "UTC")
	if err != nil || due || key != "2026-02" || start.Format("2006-01-02") != "2026-02-01" || end.Format("2006-01-02") != "2026-03-01" {
		t.Fatalf("before: %s %s %s due=%v err=%v", start, end, key, due, err)
	}
	after := before.Add(time.Minute)
	_, _, key, due, err = MonthlyDigestWindow(after, "UTC")
	if err != nil || !due || key != "2026-02" {
		t.Fatalf("after key=%s due=%v err=%v", key, due, err)
	}
}

func TestMonthlyDigestIncludesMetricsAndEscapesPagePaths(t *testing.T) {
	stats := DigestStats{NewLeads: 2, TotalVisits: 30, UniqueVisitors: 20, NewMembers: 4, TopPages: []PageCount{{Path: "/\"><script>", Count: 8}}}
	body := BuildDigestText(stats, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), DigestMonthly)
	html := BuildDigestHTML(stats, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), DigestMonthly)
	for _, want := range []string{"Toplam ziyaret: 30", "Tekil ziyaretçi: 20", "Yeni üye: 4", "/\"><script>: 8"} {
		if !strings.Contains(body, want) {
			t.Fatalf("text lacks %q: %s", want, body)
		}
	}
	if strings.Contains(html, "<script>") || !strings.Contains(html, "&lt;script&gt;") || strings.Count(html, "<ul>") != strings.Count(html, "</ul>") {
		t.Fatalf("invalid or unsafe HTML: %s", html)
	}
}

func TestWeeklyDigestHTMLEscapesReferrers(t *testing.T) {
	html := BuildWeeklyDigestHTML(WeeklyDigestStats{NewLeads: 2, TopReferrers: []ReferrerCount{{Host: "<script>bad</script>", Count: 3}}}, time.Now(), time.Now())
	if strings.Contains(html, "<script>bad") || !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf("unsafe digest HTML: %s", html)
	}
}

func TestDeliverDueDigestsSendsOnlyToActiveOptedInUsers(t *testing.T) {
	now := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	preferences := []DigestPreference{{UserID: "active", Timezone: "Europe/Istanbul"}, {UserID: "disabled", Timezone: "UTC"}, {UserID: "unknown-zone", Timezone: "Mars/Olympus"}}
	users := []DigestUser{{ID: "active", Email: "active@example.test", Active: true}, {ID: "disabled", Email: "disabled@example.test", Active: false}}
	summarized, delivered := 0, 0
	err := DeliverDueDigests(context.Background(), now, preferences, users, func(_ context.Context, start, end time.Time) (WeeklyDigestStats, error) {
		summarized++
		if end.Sub(start) != 7*24*time.Hour {
			t.Fatalf("unexpected UTC reporting window %s..%s", start, end)
		}
		return WeeklyDigestStats{NewLeads: 3}, nil
	}, func(_ context.Context, user DigestUser, week, text, html string) error {
		delivered++
		if user.ID != "active" || week != "2026-09-21" || !strings.Contains(text, "3") || !strings.Contains(html, "3") {
			t.Fatalf("unexpected digest recipient=%+v week=%s", user, week)
		}
		return nil
	})
	if err != nil || summarized != 1 || delivered != 1 {
		t.Fatalf("err=%v summarized=%d delivered=%d", err, summarized, delivered)
	}
}
