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
		return "Yayında"
	case "draft":
		return "Taslak"
	case "scheduled":
		if info.ScheduledAt != nil {
			return "Zamanlandı · " + info.ScheduledAt.UTC().Format("02 Jan 15:04 UTC")
		}
		return "Zamanlandı"
	case "archived", "retracted":
		return "Arşivlendi"
	default:
		return "Diğer"
	}
}

func statusPillScheduledAt(info StatusInfo) string {
	if info.Value != "scheduled" || info.ScheduledAt == nil {
		return ""
	}
	return info.ScheduledAt.UTC().Format(time.RFC3339)
}
