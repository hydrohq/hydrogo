package transport

import (
	"fmt"
	"time"
)

var (
	turkishMonths   = [...]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}
	turkishWeekdays = [...]string{"Pazar", "Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi"}
)

// turkishDate formats t as "14 Ekim 2026", optionally followed by the weekday
// ("14 Ekim 2026, Çarşamba"). A zero time yields an empty string.
func turkishDate(t time.Time, withWeekday bool) string {
	if t.IsZero() {
		return ""
	}
	date := fmt.Sprintf("%d %s %d", t.Day(), turkishMonths[t.Month()-1], t.Year())
	if withWeekday {
		date += ", " + turkishWeekdays[t.Weekday()]
	}
	return date
}
