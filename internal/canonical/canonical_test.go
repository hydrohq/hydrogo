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

func TestLocaleDirectionUsesSafePrimarySubtag(t *testing.T) {
	for _, test := range []struct {
		locale, want string
	}{
		{"ar", "rtl"}, {"fa-IR", "rtl"}, {"he_IL", "rtl"},
		{"tr", "ltr"}, {"en-US", "ltr"}, {"unknown", "ltr"}, {"", "ltr"},
	} {
		if got := Direction(test.locale); got != test.want {
			t.Errorf("Direction(%q) = %q, want %q", test.locale, got, test.want)
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

func FuzzPath(f *testing.F) {
	seeds := []string{
		"/",
		"home",
		"/hakkimizda/",
		"café",
		"cafe\u0301",
		"/docs/%C3%BCr\u00fcn/",
		"%zz",
		"a%00b",
		"a\\b",
		"../secrets",
		"//multiple//slashes",
		"",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		res, err := Path(input)
		if err == nil {
			if len(res) > 0 && (res[0] == '/' || res[len(res)-1] == '/') {
				t.Fatalf("Path(%q) produced unstripped slashes: %q", input, res)
			}
		}
	})
}

func FuzzEmail(f *testing.F) {
	seeds := []string{
		"admin@hydrogo.io",
		"  Admin@HydroGo.IO ",
		"e\u0301xample@example.com",
		"",
		"invalid-email",
		"test@sub.domain.co.uk",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_ = Email(input)
	})
}
