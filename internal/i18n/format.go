// Package i18n contains bounded, locale-aware presentation helpers. Domain
// values remain locale-neutral; callers choose the user's locale at render
// time.
package i18n

import (
	"fmt"
	"strings"
	"time"
	"unicode"

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

// NumberCents formats an integer-cent amount without converting through
// float64, preserving exact financial values while applying locale grouping
// and decimal separators.
func NumberCents(locale string, cents int64) string {
	negative := cents < 0
	var absolute uint64
	if negative {
		absolute = uint64(-(cents + 1)) + 1 // #nosec G115 -- sign handled explicitly; the quotient of a uint64 by 100 fits int64
	} else {
		absolute = uint64(cents)
	}
	whole := absolute / 100
	minor := absolute % 100
	return formatCentsNumber(locale, int64(whole), minor, negative) // #nosec G115 -- sign handled explicitly; the quotient of a uint64 by 100 fits int64
}

func formatCentsNumber(locale string, whole int64, minor uint64, negative bool) string {
	// Number accepts int64 exactly; only the two-digit fraction is appended.
	wholeText := message.NewPrinter(tag(locale)).Sprintf("%v", number.Decimal(whole))
	separator := "."
	for _, r := range message.NewPrinter(tag(locale)).Sprintf("%v", number.Decimal(1.1)) {
		if !unicode.IsDigit(r) {
			separator = string(r)
			break
		}
	}
	if negative {
		wholeText = "-" + wholeText
	}
	return wholeText + separator + fmt.Sprintf("%02d", minor)
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
