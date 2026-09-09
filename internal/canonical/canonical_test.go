package canonical

import (
	"errors"
	"testing"
)

func TestEmailCanonicalizationIsUnicodeStable(t *testing.T) {
	if got, want := Email("  Admin@HydroGo.IO "), "admin@hydrogo.io"; got != want {
		t.Fatalf("Email() = %q, want %q", got, want)
	}
	if got, want := Email("e\u0301xample@example.com"), "éxample@example.com"; got != want {
		t.Fatalf("Email() NFC = %q, want %q", got, want)
	}
}

func TestPathCanonicalization(t *testing.T) {
	tests := map[string]string{
		" /Hakk\u0131zda/ ":     "hakkızda",
		"caf\u00e9":             "café",
		"cafe\u0301":            "café",
		"/docs/%C3%BCr\u00fcn/": "docs/ürün",
	}
	for input, want := range tests {
		got, err := Path(input)
		if err != nil {
			t.Fatalf("Path(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("Path(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPathRejectsMalformedOrUnsafeInput(t *testing.T) {
	for _, input := range []string{"%zz", "a%00b", "a\\b"} {
		if _, err := Path(input); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("Path(%q) error = %v, want ErrInvalidPath", input, err)
		}
	}
}
