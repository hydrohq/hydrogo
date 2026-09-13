package security

import (
	"net/url"
	"strings"
	"unicode"
)

// SafeReturnPath accepts only an in-origin absolute path. URL parsing is
// intentional here: encoded slashes and browser-normalized backslashes must
// not bypass a raw-string prefix check and become an open redirect.
func SafeReturnPath(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\\\r\n") || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return fallback
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fallback
		}
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(parsed.Path, "//") {
		return fallback
	}
	return value
}
