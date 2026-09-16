package site

import "testing"

func TestResolvePublicHomePath(t *testing.T) {
	fallback := "/public/pages/en/demo/html"
	cases := []struct {
		name     string
		settings []Setting
		want     string
	}{
		{name: "public internal path", settings: []Setting{{Key: "site.home_path", Value: "/public/pages/en/home/html", Public: true}}, want: "/public/pages/en/home/html"},
		{name: "private ignored", settings: []Setting{{Key: "site.home_path", Value: "/private", Public: false}}, want: fallback},
		{name: "external ignored", settings: []Setting{{Key: "site.home_path", Value: "https://evil.example", Public: true}}, want: fallback},
		{name: "network path ignored", settings: []Setting{{Key: "site.home_path", Value: "//evil.example", Public: true}}, want: fallback},
		{name: "backslash path ignored", settings: []Setting{{Key: "site.home_path", Value: "/\\evil.example", Public: true}}, want: fallback},
		{name: "encoded network path ignored", settings: []Setting{{Key: "site.home_path", Value: "/%2F%2Fevil.example", Public: true}}, want: fallback},
		{name: "oversized path ignored", settings: []Setting{{Key: "site.home_path", Value: "/" + string(make([]byte, 2048)), Public: true}}, want: fallback},
		{name: "newline ignored", settings: []Setting{{Key: "site.home_path", Value: "/home\nnext", Public: true}}, want: fallback},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolvePublicHomePath(tc.settings, fallback); got != tc.want {
				t.Fatalf("ResolvePublicHomePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateSettingRejectsUnsafePublicHomePath(t *testing.T) {
	for _, value := range []string{"https://evil.example", "//evil.example", "/%2F%2Fevil.example", "/\\evil.example", "/home\nnext"} {
		if err := ValidateSetting(Setting{Key: "site.home_path", Value: value, Public: true}); err != ErrInvalidSetting {
			t.Fatalf("ValidateSetting(%q) = %v, want ErrInvalidSetting", value, err)
		}
	}
	if err := ValidateSetting(Setting{Key: "site.home_path", Value: "/public/pages/en/home/html", Public: true}); err != nil {
		t.Fatalf("ValidateSetting(valid home path) = %v", err)
	}
}

func TestValidateIdempotencyKey(t *testing.T) {
	if err := ValidateIdempotencyKey("0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"short", " leading0123456789", "0123456789abcdef\n"} {
		if err := ValidateIdempotencyKey(key); err != ErrInvalidIdempotencyKey {
			t.Fatalf("key %q error = %v", key, err)
		}
	}
}
