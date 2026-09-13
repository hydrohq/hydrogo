package security

import (
	"strings"
	"testing"
)

func TestSafeReturnPathRejectsNormalizedExternalTargets(t *testing.T) {
	const fallback = "/member/account"
	for _, value := range []string{"https://evil.example", "//evil.example", "/%2f%2fevil.example", "/\\evil.example", "/a\r\nLocation: https://evil.example"} {
		if got := SafeReturnPath(value, fallback); got != fallback {
			t.Fatalf("SafeReturnPath(%q) = %q, want fallback", value, got)
		}
	}
	for _, value := range []string{"/member/account", "/shop/cart?step=1", "/public/news/tr/html#latest"} {
		if got := SafeReturnPath(value, fallback); got != value {
			t.Fatalf("SafeReturnPath(%q) = %q, want original", value, got)
		}
	}
}

func FuzzSafeReturnPath(f *testing.F) {
	for _, seed := range []string{"/", "/member/account", "//evil.example", "/%2f%2fevil", "/\\evil", "https://evil.example", "/shop?next=1"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		got := SafeReturnPath(value, "/fallback")
		if got == "/fallback" {
			return
		}
		if got == "" || got[0] != '/' || strings.HasPrefix(got, "//") || strings.ContainsAny(got, "\\\r\n") {
			t.Fatalf("unsafe return path accepted: %q", got)
		}
	})
}
