// Package i18n contains bounded, locale-aware presentation helpers. Domain
// values remain locale-neutral; callers choose the user's locale at render
// time.
package i18n

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

func tag(locale string) language.Tag {
	t, err := language.Parse(strings.TrimSpace(locale))
	if err != nil {
		return language.English
	}
	return t
}

// Number formats grouping and decimals according to locale.
func Number(locale string, value float64) string {
	return message.NewPrinter(tag(locale)).Sprintf("%v", number.Decimal(value))
}

// Currency formats an ISO-4217 amount. Unknown codes are rejected instead of
// silently displaying a misleading currency.
func Currency(locale, code string, value float64) (string, error) {
	unit, err := currency.ParseISO(strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return "", fmt.Errorf("invalid currency %q: %w", code, err)
	}
	return message.NewPrinter(tag(locale)).Sprintf("%v", unit.Amount(value)), nil
}

// Date uses stable, readable order per locale while preserving the instant.
func Date(locale string, value time.Time) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "tr") {
		return value.Format("02.01.2006")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "ja") {
		return value.Format("2006/01/02")
	}
	return value.Format("01/02/2006")
}
