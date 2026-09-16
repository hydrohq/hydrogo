package i18n

import (
	"errors"
	"strings"
	"unicode"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

var ErrInvalidLocaleFallback = errors.New("invalid locale fallback configuration")

// LocaleFallbackChain returns the requested locale, its primary-language
// fallback, and the configured default when those locales are supported. The
// result is deterministic and contains no duplicates.
func LocaleFallbackChain(requested string, supported []string, defaultLocale string) ([]string, error) {
	if len(supported) > 128 {
		return nil, ErrInvalidLocaleFallback
	}
	supportedSet := make(map[string]struct{}, len(supported))
	for _, raw := range supported {
		locale, ok := normalizeLocale(raw)
		if !ok {
			return nil, ErrInvalidLocaleFallback
		}
		if _, exists := supportedSet[locale]; exists {
			return nil, ErrInvalidLocaleFallback
		}
		supportedSet[locale] = struct{}{}
	}
	requestedLocale, ok := normalizeLocale(requested)
	if !ok {
		return nil, ErrInvalidLocaleFallback
	}
	defaultValue, ok := normalizeLocale(defaultLocale)
	if !ok {
		return nil, ErrInvalidLocaleFallback
	}
	result := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	add := func(locale string) {
		if _, supported := supportedSet[locale]; !supported {
			return
		}
		if _, exists := seen[locale]; exists {
			return
		}
		seen[locale] = struct{}{}
		result = append(result, locale)
	}
	add(requestedLocale)
	if index := strings.IndexByte(requestedLocale, '-'); index > 0 {
		add(requestedLocale[:index])
	}
	add(defaultValue)
	return result, nil
}

func normalizeLocale(value string) (string, bool) {
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", false
		}
	}
	locale := canonical.Locale(value)
	if locale == "" || len(locale) > 64 || strings.ContainsAny(locale, "/\\") {
		return "", false
	}
	return locale, true
}
