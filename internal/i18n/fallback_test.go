package i18n

import (
	"reflect"
	"testing"
)

func TestLocaleFallbackChainPrefersExactThenPrimaryThenDefault(t *testing.T) {
	got, err := LocaleFallbackChain("tr-TR", []string{"en", "tr", "tr-tr"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"tr-tr", "tr", "en"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback chain = %#v, want %#v", got, want)
	}
}

func TestLocaleFallbackChainSkipsUnsupportedAndDuplicates(t *testing.T) {
	got, err := LocaleFallbackChain("de-DE", []string{"tr", "en"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"en"}) {
		t.Fatalf("fallback chain = %#v", got)
	}
}

func TestLocaleFallbackChainRejectsDuplicateOrUnsafeConfiguration(t *testing.T) {
	if _, err := LocaleFallbackChain("tr", []string{"tr", "tr"}, "en"); err != ErrInvalidLocaleFallback {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := LocaleFallbackChain("tr\n", []string{"tr"}, "tr"); err != ErrInvalidLocaleFallback {
		t.Fatalf("unsafe requested error = %v", err)
	}
}
