package site

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidAppearance = errors.New("invalid appearance settings")
	hexColorPattern      = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// Appearance contains the small set of public brand tokens that Core can
// apply without depending on a theme-specific CSS framework.
type Appearance struct {
	PrimaryColor        string
	AccentColor         string
	DarkPrimaryColor    string
	DarkAccentColor     string
	FontFamily          string
	LogoURL             string
	DarkLogoURL         string
	FaviconURL          string
	SocialImageURL      string
	LoginBackgroundURL  string
	LoginOverlayOpacity float64
}

var appearanceFonts = map[string]string{
	"system": `system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`,
	"inter":  `Inter, system-ui, sans-serif`,
	"lato":   `Lato, system-ui, sans-serif`,
	"roboto": `Roboto, system-ui, sans-serif`,
}

func (a *Appearance) Normalize() error {
	if a == nil {
		return ErrInvalidAppearance
	}
	a.PrimaryColor = strings.ToUpper(strings.TrimSpace(a.PrimaryColor))
	a.AccentColor = strings.ToUpper(strings.TrimSpace(a.AccentColor))
	a.DarkPrimaryColor = strings.ToUpper(strings.TrimSpace(a.DarkPrimaryColor))
	a.DarkAccentColor = strings.ToUpper(strings.TrimSpace(a.DarkAccentColor))
	a.FontFamily = strings.ToLower(strings.TrimSpace(a.FontFamily))
	a.LogoURL = strings.TrimSpace(a.LogoURL)
	a.DarkLogoURL = strings.TrimSpace(a.DarkLogoURL)
	a.FaviconURL = strings.TrimSpace(a.FaviconURL)
	a.SocialImageURL = strings.TrimSpace(a.SocialImageURL)
	a.LoginBackgroundURL = strings.TrimSpace(a.LoginBackgroundURL)
	if a.LoginOverlayOpacity < 0 || a.LoginOverlayOpacity > .8 || math.IsNaN(a.LoginOverlayOpacity) || math.IsInf(a.LoginOverlayOpacity, 0) {
		return ErrInvalidAppearance
	}
	if a.PrimaryColor == "" {
		a.PrimaryColor = "#155EEF"
	}
	if a.AccentColor == "" {
		a.AccentColor = "#155EEF"
	}
	if a.DarkPrimaryColor == "" {
		a.DarkPrimaryColor = a.PrimaryColor
	}
	if a.DarkAccentColor == "" {
		a.DarkAccentColor = a.AccentColor
	}
	if a.LogoURL == "" && a.DarkLogoURL != "" {
		a.LogoURL = a.DarkLogoURL
	}
	if a.FontFamily == "" {
		a.FontFamily = "system"
	}
	if _, ok := appearanceFonts[a.FontFamily]; !ok || !hexColorPattern.MatchString(a.PrimaryColor) || !hexColorPattern.MatchString(a.AccentColor) || !hexColorPattern.MatchString(a.DarkPrimaryColor) || !hexColorPattern.MatchString(a.DarkAccentColor) {
		return ErrInvalidAppearance
	}
	for _, asset := range []string{a.LogoURL, a.DarkLogoURL, a.FaviconURL, a.SocialImageURL, a.LoginBackgroundURL} {
		if !ValidAppearanceAssetURL(asset) {
			return ErrInvalidAppearance
		}
	}
	if ContrastRatio("#FFFFFF", a.PrimaryColor) < 4.5 || ContrastRatio("#FFFFFF", a.AccentColor) < 4.5 || ContrastRatio("#FFFFFF", a.DarkPrimaryColor) < 4.5 || ContrastRatio("#FFFFFF", a.DarkAccentColor) < 4.5 {
		return ErrInvalidAppearance
	}
	return nil
}

// ValidAppearanceAssetURL accepts only same-origin media or theme files so
// saved branding cannot introduce remote tracking or CSS/HTML injection.
func ValidAppearanceAssetURL(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != value || strings.ContainsAny(value, "\\\r\n\x00\"'()") {
		return false
	}
	if !strings.HasPrefix(value, "/media/") && !strings.HasPrefix(value, "/theme/static/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	return true
}

// ContrastRatio returns the WCAG relative-luminance ratio. Invalid colors
// return zero so they cannot accidentally pass the AA threshold.
func ContrastRatio(foreground, background string) float64 {
	fg, ok := relativeLuminance(foreground)
	if !ok {
		return 0
	}
	bg, ok := relativeLuminance(background)
	if !ok {
		return 0
	}
	if fg < bg {
		fg, bg = bg, fg
	}
	return (fg + 0.05) / (bg + 0.05)
}

func relativeLuminance(value string) (float64, bool) {
	if !hexColorPattern.MatchString(value) {
		return 0, false
	}
	channel := func(pair string) float64 {
		parsed, _ := strconv.ParseUint(pair, 16, 8)
		v := float64(parsed) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	r, g, b := channel(value[1:3]), channel(value[3:5]), channel(value[5:7])
	return 0.2126*r + 0.7152*g + 0.0722*b, true
}

// CSS returns only validated values and a fixed font allowlist, safe to place
// in a nonce-bearing style element in the rendered page head.
func (a Appearance) CSS() (string, error) {
	if err := a.Normalize(); err != nil {
		return "", err
	}
	font := appearanceFonts[a.FontFamily]
	css := fmt.Sprintf(`:root{--brand:%s;--accent:%s;--accent-cyan:%s;--font-sans:%s}@media(prefers-color-scheme:dark){:root{--brand:%s;--accent:%s;--accent-cyan:%s;--font-sans:%s}}`, a.PrimaryColor, a.PrimaryColor, a.AccentColor, font, a.DarkPrimaryColor, a.DarkPrimaryColor, a.DarkAccentColor, font)
	if a.LogoURL != "" {
		darkLogo := a.DarkLogoURL
		if darkLogo == "" {
			darkLogo = a.LogoURL
		}
		css += fmt.Sprintf(`.news-logo .logo-mark,.news-logo .logo-text{display:none}.news-logo:before{content:"";display:block;width:160px;height:48px;background:url("%s") center/contain no-repeat}@media(prefers-color-scheme:dark){.news-logo:before{background-image:url("%s")}}`, a.LogoURL, darkLogo)
	}
	return css, nil
}

// AppearanceFromSettings extracts public site appearance fields, ignoring
// unrelated keys and falling back to the product defaults.
func AppearanceFromSettings(settings []Setting) Appearance {
	a := Appearance{FontFamily: "system", LoginOverlayOpacity: .4}
	for _, item := range settings {
		switch item.Key {
		case "appearance.primary_color":
			a.PrimaryColor = item.Value
		case "appearance.accent_color":
			a.AccentColor = item.Value
		case "appearance.dark_primary_color":
			a.DarkPrimaryColor = item.Value
		case "appearance.dark_accent_color":
			a.DarkAccentColor = item.Value
		case "appearance.font_family":
			a.FontFamily = item.Value
		case "appearance.logo_url":
			a.LogoURL = item.Value
		case "appearance.dark_logo_url":
			a.DarkLogoURL = item.Value
		case "appearance.favicon_url":
			a.FaviconURL = item.Value
		case "appearance.social_image_url":
			a.SocialImageURL = item.Value
		case "appearance.login_background_url":
			a.LoginBackgroundURL = item.Value
		case "appearance.login_overlay_opacity":
			if parsed, err := strconv.ParseFloat(item.Value, 64); err == nil {
				a.LoginOverlayOpacity = parsed
			}
		}
	}
	if a.Normalize() != nil {
		return Appearance{PrimaryColor: "#155EEF", AccentColor: "#155EEF", DarkPrimaryColor: "#155EEF", DarkAccentColor: "#155EEF", FontFamily: "system", LoginOverlayOpacity: .4}
	}
	return a
}
