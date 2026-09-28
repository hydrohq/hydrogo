package ui

import (
	"fmt"
	"time"
)

type MediaGarbageItem struct {
	ID        string
	Filename  string
	SizeBytes int64
	CreatedAt string
}

func formatMediaGarbageSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(bytes) / 1024
	for _, unit := range units {
		if value < 1024 || unit == units[len(units)-1] {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
		value /= 1024
	}
	return ""
}

func mediaGarbageDate(value time.Time) string { return value.UTC().Format("2006-01-02 15:04 UTC") }
