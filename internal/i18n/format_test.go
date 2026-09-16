package i18n

import (
	"strings"
	"testing"
	"time"
)

func TestLocaleFormatting(t *testing.T) {
	date := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		locale   string
		wantDate string
		wantNum  []string
	}{
		{name: "Turkish", locale: "tr-TR", wantDate: "15.09.2026", wantNum: []string{"1", ","}},
		{name: "Japanese", locale: "ja-JP", wantDate: "2026/09/15", wantNum: []string{"1"}},
		{name: "English", locale: "en-US", wantDate: "09/15/2026", wantNum: []string{"1", "."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Date(tt.locale, date); got != tt.wantDate {
				t.Fatalf("date(%q) = %q, want %q", tt.locale, got, tt.wantDate)
			}
			got := Number(tt.locale, 1234567.89)
			for _, want := range tt.wantNum {
				if !strings.Contains(got, want) {
					t.Fatalf("number(%q) = %q, missing %q", tt.locale, got, want)
				}
			}
		})
	}

	formatted, err := Currency("en-US", "USD", 12.5)
	if err != nil || !strings.Contains(formatted, "USD") || !strings.Contains(formatted, "12.50") {
		t.Fatalf("currency=%q err=%v", formatted, err)
	}
	trCurrency, err := Currency("tr-TR", "TRY", 1234.5)
	if err != nil || !strings.Contains(trCurrency, "1.234,50") {
		t.Fatalf("Turkish currency=%q err=%v", trCurrency, err)
	}
	if _, err := Currency("en-US", "NOT-A-CURRENCY", 1); err == nil {
		t.Fatal("invalid currency accepted")
	}
}

func TestLocaleFormattingFallsBackSafely(t *testing.T) {
	date := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	if got := Date("not a locale", date); got != "09/15/2026" {
		t.Fatalf("invalid locale date = %q", got)
	}
	if got := Number("not a locale", 1234.5); got == "" {
		t.Fatal("invalid locale produced empty number")
	}
	if got, err := Currency("not a locale", "USD", 12.5); err != nil || got == "" {
		t.Fatalf("invalid locale currency=%q err=%v", got, err)
	}
}
