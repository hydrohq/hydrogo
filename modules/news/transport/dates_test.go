package transport

import (
	"testing"
	"time"
)

func TestTurkishDate(t *testing.T) {
	day := time.Date(2026, time.October, 14, 9, 0, 0, 0, time.UTC)
	if got := turkishDate(day, true); got != "14 Ekim 2026, Çarşamba" {
		t.Fatalf("with weekday = %q", got)
	}
	if got := turkishDate(day, false); got != "14 Ekim 2026" {
		t.Fatalf("without weekday = %q", got)
	}
	if got := turkishDate(time.Time{}, true); got != "" {
		t.Fatalf("zero time = %q", got)
	}
}
