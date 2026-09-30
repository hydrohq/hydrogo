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

// Slugify creates an ASCII URL segment, including Turkish-specific letters.
func Slugify(value string) string {
	value = strings.NewReplacer("ı", "i", "İ", "i", "ß", "ss", "æ", "ae", "œ", "oe").Replace(strings.ToLower(norm.NFC.String(value)))
	value = norm.NFD.String(value)
	var out strings.Builder
	separator := false
	for _, r := range value {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if separator && out.Len() > 0 {
				out.WriteByte('-')
			}
			separator = false
			out.WriteRune(r)
		} else {
			separator = true
		}
	}
	return strings.Trim(out.String(), "-")
}

// Direction returns the safe document direction for a locale. Unknown and
// malformed locale tags intentionally default to ltr; callers must not infer
// direction from user-provided markup.
func Direction(locale string) string {
	primary := strings.ToLower(strings.TrimSpace(locale))
	if index := strings.IndexAny(primary, "-_"); index >= 0 {
		primary = primary[:index]
	}
	switch primary {
	case "ar", "dv", "fa", "he", "ku", "ps", "sd", "ur", "yi":
		return "rtl"
	default:
		return "ltr"
	}
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
