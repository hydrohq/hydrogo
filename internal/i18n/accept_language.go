package i18n

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

// ResolveAcceptLanguage returns the highest-ranked supported locale from an
// Accept-Language header, falling back to defaultLocale when none match.
func ResolveAcceptLanguage(header string, supported []string, defaultLocale string) string {
	fallback := supportedLocale(defaultLocale, supported)
	if fallback == "" {
		return ""
	}
	type preference struct {
		locale string
		q      float64
		order  int
	}
	preferences := make([]preference, 0, 8)
	for order, item := range strings.Split(header, ",") {
		parts := strings.Split(item, ";")
		locale := canonical.Locale(strings.TrimSpace(parts[0]))
		if locale == "" || locale == "*" {
			continue
		}
		q := 1.0
		valid := true
		for _, param := range parts[1:] {
			key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil || parsed < 0 || parsed > 1 {
				valid = false
			}
			q = parsed
		}
		if valid && q > 0 {
			preferences = append(preferences, preference{locale: locale, q: q, order: order})
		}
	}
	sort.SliceStable(preferences, func(i, j int) bool { return preferences[i].q > preferences[j].q })
	for _, preference := range preferences {
		if match := supportedLocale(preference.locale, supported); match != "" {
			return match
		}
		if index := strings.IndexByte(preference.locale, '-'); index > 0 {
			if match := supportedPrimaryLocale(preference.locale[:index], supported); match != "" {
				return match
			}
			if match := supportedLocale(preference.locale[:index], supported); match != "" {
				return match
			}
		} else if match := supportedPrimaryLocale(preference.locale, supported); match != "" {
			return match
		}
	}
	return fallback
}

func supportedPrimaryLocale(primary string, supported []string) string {
	for _, candidate := range supported {
		if index := strings.IndexByte(canonical.Locale(candidate), '-'); index > 0 && canonical.Locale(candidate[:index]) == primary {
			return candidate
		}
	}
	return ""
}

func supportedLocale(locale string, supported []string) string {
	locale = canonical.Locale(locale)
	if locale == "" {
		return ""
	}
	for _, candidate := range supported {
		if canonical.Locale(candidate) == locale {
			return candidate
		}
	}
	return ""
}
