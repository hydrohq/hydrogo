package analytics

import "testing"

func TestSafeConsentReturnPathAcceptsOnlyLocalPaths(t *testing.T) {
	for _, input := range []string{"/about", "/public/page?locale=en", "/"} {
		if got := safeConsentReturnPath(input); got != input {
			t.Errorf("safeConsentReturnPath(%q) = %q", input, got)
		}
	}
	for _, input := range []string{"", "//evil.test", "https://evil.test", "/\\evil.test", "/\r\nLocation: https://evil.test"} {
		if got := safeConsentReturnPath(input); got != "" {
			t.Errorf("safeConsentReturnPath(%q) = %q, want empty", input, got)
		}
	}
}
