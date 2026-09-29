package site

import (
	"strings"
	"testing"
)

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

func TestContactWidgetSettingsRoundTripAndValidation(t *testing.T) {
	widget := ContactWidget{Enabled: true, Phone: "+90 212 555 0100", WhatsAppNumber: "+90 555 123 4567", Message: "Merhaba", Position: "bottom-left", MobileOnly: true}
	settings := widget.Settings()
	if !ContactWidgetConfigured(settings) {
		t.Fatal("saved widget should be considered configured")
	}
	if got := ContactWidgetFromSettings(settings); got != widget {
		t.Fatalf("round trip = %#v, want %#v", got, widget)
	}
	if err := widget.Validate(); err != nil {
		t.Fatalf("valid widget rejected: %v", err)
	}
	if ContactWidgetConfigured(nil) {
		t.Fatal("missing settings should allow legacy fallback")
	}
	for _, invalid := range []ContactWidget{
		{Enabled: true, Position: "bottom-right"},
		{Enabled: true, Phone: "javascript:alert(1)", Position: "bottom-right"},
		{Position: "center", Phone: "+90 212 555 0100"},
		{Message: strings.Repeat("x", 501), Position: "bottom-right"},
	} {
		if err := invalid.Validate(); err != ErrInvalidSetting {
			t.Fatalf("invalid widget %#v error = %v", invalid, err)
		}
	}
}

func TestNormalizeSocialProfilesAcceptsOnlyPlatformHosts(t *testing.T) {
	profiles, err := NormalizeSocialProfiles([]SocialProfile{{Platform: "instagram", URL: "https://www.instagram.com/hydrogo"}, {Platform: "github", URL: "https://github.com/hydrogo"}})
	if err != nil || len(profiles) != 2 {
		t.Fatalf("valid profiles = %#v, error=%v", profiles, err)
	}
	for _, invalid := range [][]SocialProfile{
		{{Platform: "instagram", URL: "https://instagram.attacker.test/hydrogo"}},
		{{Platform: "youtube", URL: "http://youtube.com/hydrogo"}},
		{{Platform: "github", URL: "https://user:secret@github.com/hydrogo"}},
		{{Platform: "github", URL: "https://github.com:444/hydrogo"}},
		{{Platform: "myspace", URL: "https://myspace.com/hydrogo"}},
		{{Platform: "github", URL: "https://github.com/a"}, {Platform: "github", URL: "https://github.com/b"}},
	} {
		if _, err := NormalizeSocialProfiles(invalid); err != ErrInvalidSetting {
			t.Fatalf("invalid profiles %#v error=%v", invalid, err)
		}
	}
}

func TestValidateSubmissionLocationMetadata(t *testing.T) {
	base := Submission{ID: "submission-1", FormKey: "contact", Email: "visitor@example.test", SourceIPHash: "hash", Payload: `{}`}
	for _, tc := range []struct {
		name  string
		code  string
		city  string
		local bool
		valid bool
	}{
		{name: "country and city", code: "TR", city: "Istanbul", valid: true},
		{name: "local network", local: true, valid: true},
		{name: "lowercase code", code: "tr"},
		{name: "local with country", code: "TR", local: true},
		{name: "city too long", code: "TR", city: strings.Repeat("x", 129)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := base
			value.LocationCountryCode, value.LocationCity, value.LocationLocal = tc.code, tc.city, tc.local
			err := ValidateSubmission(value)
			if (err == nil) != tc.valid {
				t.Fatalf("ValidateSubmission() error=%v valid=%v", err, tc.valid)
			}
		})
	}
}
