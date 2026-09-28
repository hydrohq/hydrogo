package adminstudio

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"
	_ "time/tzdata"
)

type ReferrerCount struct {
	Host  string
	Count int64
}
type WeeklyDigestStats struct {
	NewLeads        int64
	UnresolvedLeads int64
	TopReferrers    []ReferrerCount
}
type DigestPreference struct{ UserID, Timezone, LastSentWeek string }
type DigestUser struct {
	ID, Email string
	Active    bool
}
type DigestStatsSource func(context.Context, time.Time, time.Time) (WeeklyDigestStats, error)
type DigestDeliver func(context.Context, DigestUser, string, string, string) error

// DeliverDueDigests maps preferences to enabled accounts, summarizes one
// local calendar week per timezone and delegates durable, idempotent delivery.
func DeliverDueDigests(ctx context.Context, now time.Time, preferences []DigestPreference, users []DigestUser, summarize DigestStatsSource, deliver DigestDeliver) error {
	if ctx == nil || summarize == nil || deliver == nil {
		return fmt.Errorf("weekly digest dependencies are required")
	}
	userByID := make(map[string]DigestUser, len(users))
	for _, user := range users {
		if user.Active && user.Email != "" {
			userByID[user.ID] = user
		}
	}
	statsByWindow := map[string]WeeklyDigestStats{}
	for _, preference := range preferences {
		user, ok := userByID[preference.UserID]
		if !ok {
			continue
		}
		start, end, key, due, err := WeeklyDigestWindow(now, preference.Timezone)
		if err != nil {
			return err
		}
		if !due || preference.LastSentWeek == key {
			continue
		}
		windowKey := start.UTC().Format(time.RFC3339) + "/" + end.UTC().Format(time.RFC3339)
		stats, ok := statsByWindow[windowKey]
		if !ok {
			stats, err = summarize(ctx, start, end)
			if err != nil {
				return err
			}
			statsByWindow[windowKey] = stats
		}
		if err := deliver(ctx, user, key, BuildWeeklyDigestText(stats, start, end), BuildWeeklyDigestHTML(stats, start, end)); err != nil {
			return err
		}
	}
	return nil
}

// WeeklyDigestWindow returns the previous complete Monday-to-Monday period
// once local Monday 09:00 has passed. Calendar arithmetic preserves DST
// transitions and the returned key makes delivery idempotent.
func WeeklyDigestWindow(now time.Time, timezone string) (start, end time.Time, weekKey string, due bool, err error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, "", false, err
	}
	local := now.In(location)
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	end = time.Date(local.Year(), local.Month(), local.Day()-daysSinceMonday, 0, 0, 0, 0, location)
	due = !local.Before(end.Add(9 * time.Hour))
	start = end.AddDate(0, 0, -7)
	weekKey = start.Format("2006-01-02")
	return start, end, weekKey, due, nil
}

func BuildWeeklyDigestText(stats WeeklyDigestStats, start, end time.Time) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Haftalık site özeti · %s – %s\n\n", start.Format("02.01.2006"), end.AddDate(0, 0, -1).Format("02.01.2006"))
	fmt.Fprintf(&out, "Yeni müşteri talepleri: %d\nYanıt bekleyen talepler: %d\n", stats.NewLeads, stats.UnresolvedLeads)
	if len(stats.TopReferrers) > 0 {
		out.WriteString("\nEn çok ziyaretçi getiren kaynaklar:\n")
		for _, item := range stats.TopReferrers {
			fmt.Fprintf(&out, "• %s: %d\n", item.Host, item.Count)
		}
	}
	return out.String()
}

func BuildWeeklyDigestHTML(stats WeeklyDigestStats, start, end time.Time) string {
	var out strings.Builder
	fmt.Fprintf(&out, "<h1>Haftalık site özeti</h1><p>%s – %s</p><ul><li>Yeni müşteri talepleri: %d</li><li>Yanıt bekleyen talepler: %d</li></ul>", template.HTMLEscapeString(start.Format("02.01.2006")), template.HTMLEscapeString(end.AddDate(0, 0, -1).Format("02.01.2006")), stats.NewLeads, stats.UnresolvedLeads)
	if len(stats.TopReferrers) > 0 {
		out.WriteString("<h2>En çok ziyaretçi getiren kaynaklar</h2><ul>")
		for _, item := range stats.TopReferrers {
			fmt.Fprintf(&out, "<li>%s: %d</li>", template.HTMLEscapeString(item.Host), item.Count)
		}
		out.WriteString("</ul>")
	}
	return out.String()
}
