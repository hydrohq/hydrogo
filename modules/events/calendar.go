package events

import (
	"fmt"
	"strings"
	"time"
)

func CalendarInvite(item Event) string {
	if item.Validate() != nil {
		return ""
	}
	uid := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, item.Slug) + "@hydrogo"
	var out strings.Builder
	out.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//HydroGo//Events//EN\r\nBEGIN:VEVENT\r\n")
	fmt.Fprintf(&out, "UID:%s\r\nDTSTAMP:%s\r\nDTSTART:%s\r\nDTEND:%s\r\nSUMMARY:%s\r\nLOCATION:%s\r\nDESCRIPTION:%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n", uid, time.Now().UTC().Format("20060102T150405Z"), item.StartsAt.UTC().Format("20060102T150405Z"), item.EndsAt.UTC().Format("20060102T150405Z"), icsText(item.Title), icsText(item.Venue), icsText(item.Summary+"\n"+item.Body))
	return out.String()
}

func icsText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, ";", "\\;")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, "\r\n", "\\n")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return strings.ReplaceAll(value, "\r", "\\n")
}
