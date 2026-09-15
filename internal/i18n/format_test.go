package i18n

import (
	"strings"
	"testing"
	"time"
)

func TestLocaleFormatting(t *testing.T) {
	if got := Number("tr-TR", 1234567.89); !strings.Contains(got, "1") || !strings.Contains(got, ",") {
		t.Fatalf("Turkish number = %q", got)
	}
	if got := Date("tr-TR", time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)); got != "15.09.2026" {
		t.Fatalf("date = %q", got)
	}
	formatted, err := Currency("en-US", "USD", 12.5)
	if err != nil || !strings.Contains(formatted, "12.50") {
		t.Fatalf("currency=%q err=%v", formatted, err)
	}
	if _, err := Currency("en-US", "NOT-A-CURRENCY", 1); err == nil {
		t.Fatal("invalid currency accepted")
	}
}
