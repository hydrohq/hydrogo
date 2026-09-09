package canonical

import (
	"errors"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var ErrInvalidPath = errors.New("invalid canonical path")

// Email applies storage and lookup canonicalization. Provider-specific
// aliases are intentionally not applied here.
func Email(value string) string {
	return norm.NFC.String(strings.ToLower(strings.TrimSpace(value)))
}

func Locale(value string) string {
	return strings.Trim(norm.NFC.String(strings.ToLower(strings.TrimSpace(value))), "/")
}

// Path decodes one URL representation and returns the canonical lookup path.
// It preserves the existing multi-segment route behavior while rejecting
// malformed escapes, control characters, and backslashes.
func Path(value string) (string, error) {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", ErrInvalidPath
	}
	decoded = strings.Trim(norm.NFC.String(strings.ToLower(strings.TrimSpace(decoded))), "/")
	for _, r := range decoded {
		if unicode.IsControl(r) || r == '\\' {
			return "", ErrInvalidPath
		}
	}
	return decoded, nil
}
