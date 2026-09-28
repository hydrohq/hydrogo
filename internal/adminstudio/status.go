package adminstudio

import (
	"strings"
	"time"
)

func statusPillClass(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "published":
		return "status-pill status-pill--published"
	case "draft":
		return "status-pill status-pill--draft"
	case "scheduled":
		return "status-pill status-pill--scheduled"
	case "archived", "retracted":
		return "status-pill status-pill--archived"
	default:
		return "status-pill"
	}
}

func statusPillLabel(info StatusInfo) string {
	switch strings.ToLower(strings.TrimSpace(info.Value)) {
	case "published":
		return "Published"
	case "draft":
		return "Draft"
	case "scheduled":
		if info.ScheduledAt != nil {
			return "Scheduled " + info.ScheduledAt.UTC().Format("02 Jan 15:04 UTC")
		}
		return "Scheduled"
	case "archived", "retracted":
		return "Archived"
	default:
		return "Other"
	}
}

func statusPillScheduledAt(info StatusInfo) string {
	if info.Value != "scheduled" || info.ScheduledAt == nil {
		return ""
	}
	return info.ScheduledAt.UTC().Format(time.RFC3339)
}
