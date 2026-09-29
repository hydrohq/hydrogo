package site

import (
	"strings"
	"testing"
)

func TestContrastRatioMatchesWCAGExamples(t *testing.T) {
	if got := ContrastRatio("#FFFFFF", "#000000"); got != 21 {
		t.Fatalf("black/white contrast = %v, want 21", got)
	}
	if got := ContrastRatio("#FFFFFF", "#767676"); got < 4.5 || got > 4.6 {
		t.Fatalf("gray/white contrast = %v, want about 4.54", got)
	}
	if got := ContrastRatio("bad", "#FFFFFF"); got != 0 {
		t.Fatalf("invalid color contrast = %v, want 0", got)
	}
}

func TestAppearanceRequiresAAWhiteTextAndUsesAllowlistedCSS(t *testing.T) {
	value := Appearance{PrimaryColor: "#155eef", AccentColor: "#155eef", DarkPrimaryColor: "#155eef", DarkAccentColor: "#155eef", FontFamily: "Inter", LogoURL: "/media/brand.svg", DarkLogoURL: "/media/brand-dark.svg", FaviconURL: "/media/favicon.svg", SocialImageURL: "/media/social.webp"}
	css, err := value.CSS()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(css, `--font-sans:Inter, system-ui, sans-serif`) || !strings.Contains(css, `background:url("/media/brand.svg")`) || !strings.Contains(css, `url("/media/brand-dark.svg")`) || strings.Contains(css, "<") || strings.Contains(css, "expression") {
		t.Fatalf("unexpected generated CSS: %q", css)
	}
	invalid := Appearance{PrimaryColor: "#FFFFFF", AccentColor: "#FFFFFF", DarkPrimaryColor: "#FFFFFF", DarkAccentColor: "#FFFFFF", FontFamily: "system"}
	if err := invalid.Normalize(); err == nil {
		t.Fatal("low contrast palette accepted")
	}
	invalid = Appearance{PrimaryColor: "#155EEF;body{display:none}", AccentColor: "#155EEF", DarkPrimaryColor: "#155EEF", DarkAccentColor: "#155EEF", FontFamily: "system"}
	if err := invalid.Normalize(); err == nil {
		t.Fatal("CSS injection in color accepted")
	}
	invalid = Appearance{PrimaryColor: "#155EEF", AccentColor: "#155EEF", DarkPrimaryColor: "#155EEF", DarkAccentColor: "#155EEF", FontFamily: "system", LogoURL: "https://tracker.example/logo.svg"}
	if err := invalid.Normalize(); err == nil {
		t.Fatal("external branding asset URL accepted")
	}
}

func TestAppearanceFromSettingsFallsBackOnInvalidStoredValues(t *testing.T) {
	got := AppearanceFromSettings([]Setting{{Key: "appearance.primary_color", Value: "not-a-color", Public: true}})
	if got.PrimaryColor != "#155EEF" || got.FontFamily != "system" {
		t.Fatalf("appearance fallback = %#v", got)
	}
}

func TestLoginAppearanceIsValidatedAndLoaded(t *testing.T) {
	a := AppearanceFromSettings([]Setting{{Key: "appearance.login_background_url", Value: "/media/login.webp", Public: true}, {Key: "appearance.login_overlay_opacity", Value: "0.25", Public: true}})
	if a.LoginBackgroundURL != "/media/login.webp" || a.LoginOverlayOpacity != .25 {
		t.Fatalf("login appearance = %#v", a)
	}
	a.LoginBackgroundURL = "https://tracker.example/image"
	if a.Normalize() == nil {
		t.Fatal("external login background was accepted")
	}
	a.LoginBackgroundURL = "/media/login.webp"
	a.LoginOverlayOpacity = .9
	if a.Normalize() == nil {
		t.Fatal("out of range overlay was accepted")
	}
}
