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
type PageCount struct {
	Path  string
	Count int64
}
type DigestStats struct {
	NewLeads, UnresolvedLeads   int64
	TopReferrers                []ReferrerCount
	TotalVisits, UniqueVisitors int64
	TopPages                    []PageCount
	NewMembers                  int64
}
type WeeklyDigestStats = DigestStats
type DigestPreference struct{ UserID, Timezone, Frequency, LastSentPeriod string }
type DigestUser struct {
	ID, Email string
	Active    bool
}
type DigestStatsSource func(context.Context, time.Time, time.Time) (DigestStats, error)
type DigestDeliver func(context.Context, DigestUser, string, string, string) error
type DigestFrequency string

const (
	DigestWeekly  DigestFrequency = "weekly"
	DigestMonthly DigestFrequency = "monthly"
)

// DeliverDueDigests summarizes one local calendar period per timezone and
// delegates durable, idempotent delivery.
func DeliverDueDigests(ctx context.Context, now time.Time, preferences []DigestPreference, users []DigestUser, summarize DigestStatsSource, deliver DigestDeliver) error {
	if ctx == nil || summarize == nil || deliver == nil {
		return fmt.Errorf("digest dependencies are required")
	}
	userByID := make(map[string]DigestUser, len(users))
	for _, user := range users {
		if user.Active && user.Email != "" {
			userByID[user.ID] = user
		}
	}
	statsByWindow := map[string]DigestStats{}
	for _, preference := range preferences {
		user, ok := userByID[preference.UserID]
		if !ok {
			continue
		}
		frequency := DigestFrequency(preference.Frequency)
		if frequency == "" {
			frequency = DigestWeekly
		}
		start, end, key, due, err := DigestWindow(now, preference.Timezone, frequency)
		if err != nil {
			return err
		}
		if !due || preference.LastSentPeriod == key {
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
		if err := deliver(ctx, user, key, BuildDigestText(stats, start, end, frequency), BuildDigestHTML(stats, start, end, frequency)); err != nil {
			return err
		}
	}
	return nil
}

func DigestWindow(now time.Time, timezone string, frequency DigestFrequency) (start, end time.Time, periodKey string, due bool, err error) {
	switch frequency {
	case DigestWeekly:
		return WeeklyDigestWindow(now, timezone)
	case DigestMonthly:
		return MonthlyDigestWindow(now, timezone)
	default:
		return time.Time{}, time.Time{}, "", false, fmt.Errorf("invalid digest frequency")
	}
}

// WeeklyDigestWindow returns the previous Monday-to-Monday local week after Monday 09:00.
func WeeklyDigestWindow(now time.Time, timezone string) (start, end time.Time, weekKey string, due bool, err error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, "", false, err
	}
	local := now.In(location)
	days := (int(local.Weekday()) + 6) % 7
	end = time.Date(local.Year(), local.Month(), local.Day()-days, 0, 0, 0, 0, location)
	due = !local.Before(end.Add(9 * time.Hour))
	start = end.AddDate(0, 0, -7)
	weekKey = start.Format("2006-01-02")
	return start, end, weekKey, due, nil
}

// MonthlyDigestWindow returns the previous full local month after 09:00 on day one.
func MonthlyDigestWindow(now time.Time, timezone string) (start, end time.Time, monthKey string, due bool, err error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, "", false, err
	}
	local := now.In(location)
	end = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	due = local.Day() == 1 && !local.Before(end.Add(9*time.Hour))
	start = end.AddDate(0, -1, 0)
	monthKey = start.Format("2006-01")
	return start, end, monthKey, due, nil
}

func BuildWeeklyDigestText(stats WeeklyDigestStats, start, end time.Time) string {
	return BuildDigestText(stats, start, end, DigestWeekly)
}
func BuildDigestText(stats DigestStats, start, end time.Time, frequency DigestFrequency) string {
	label := "Haftalık site özeti"
	if frequency == DigestMonthly {
		label = "Aylık yönetici özeti"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "%s · %s – %s\n\n", label, start.Format("02.01.2006"), end.AddDate(0, 0, -1).Format("02.01.2006"))
	fmt.Fprintf(&out, "Yeni müşteri talepleri: %d\nYanıt bekleyen talepler: %d\n", stats.NewLeads, stats.UnresolvedLeads)
	if frequency == DigestMonthly {
		fmt.Fprintf(&out, "Toplam ziyaret: %d\nTekil ziyaretçi: %d\nYeni üye: %d\n", stats.TotalVisits, stats.UniqueVisitors, stats.NewMembers)
		if len(stats.TopPages) > 0 {
			out.WriteString("\nEn çok görüntülenen sayfalar:\n")
			for _, p := range stats.TopPages {
				fmt.Fprintf(&out, "• %s: %d\n", p.Path, p.Count)
			}
		}
	}
	if len(stats.TopReferrers) > 0 {
		out.WriteString("\nEn çok ziyaretçi getiren kaynaklar:\n")
		for _, r := range stats.TopReferrers {
			fmt.Fprintf(&out, "• %s: %d\n", r.Host, r.Count)
		}
	}
	return out.String()
}
func BuildWeeklyDigestHTML(stats WeeklyDigestStats, start, end time.Time) string {
	return BuildDigestHTML(stats, start, end, DigestWeekly)
}
func BuildDigestHTML(stats DigestStats, start, end time.Time, frequency DigestFrequency) string {
	label := "Haftalık site özeti"
	if frequency == DigestMonthly {
		label = "Aylık yönetici özeti"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "<h1>%s</h1><p>%s – %s</p><ul><li>Yeni müşteri talepleri: %d</li><li>Yanıt bekleyen talepler: %d</li>", label, template.HTMLEscapeString(start.Format("02.01.2006")), template.HTMLEscapeString(end.AddDate(0, 0, -1).Format("02.01.2006")), stats.NewLeads, stats.UnresolvedLeads)
	if frequency == DigestMonthly {
		fmt.Fprintf(&out, "<li>Toplam ziyaret: %d</li><li>Tekil ziyaretçi: %d</li><li>Yeni üye: %d</li>", stats.TotalVisits, stats.UniqueVisitors, stats.NewMembers)
	}
	out.WriteString("</ul>")
	if frequency == DigestMonthly && len(stats.TopPages) > 0 {
		out.WriteString("<h2>En çok görüntülenen sayfalar</h2><ol>")
		for _, p := range stats.TopPages {
			fmt.Fprintf(&out, "<li>%s: %d</li>", template.HTMLEscapeString(p.Path), p.Count)
		}
		out.WriteString("</ol>")
	}
	if len(stats.TopReferrers) > 0 {
		out.WriteString("<h2>En çok ziyaretçi getiren kaynaklar</h2><ul>")
		for _, r := range stats.TopReferrers {
			fmt.Fprintf(&out, "<li>%s: %d</li>", template.HTMLEscapeString(r.Host), r.Count)
		}
		out.WriteString("</ul>")
	}
	return out.String()
}
