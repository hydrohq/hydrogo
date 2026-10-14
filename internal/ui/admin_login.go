package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// RenderAdminLogin renders the clean high-performance login screen for HydroGo.
func RenderAdminLogin(w io.Writer, csrfToken, returnTo string) error {
	return RenderAdminLoginWithAppearance(w, csrfToken, returnTo, site.Appearance{})
}

// RenderAdminLoginWithAppearance applies validated site appearance to the public admin login screen.
func RenderAdminLoginWithAppearance(w io.Writer, csrfToken, returnTo string, appearance site.Appearance) error {
	return RenderAdminLoginWithLocaleAndAppearance(w, csrfToken, returnTo, "tr", appearance)
}

// RenderAdminLoginWithLocaleAndAppearance renders the public login in a bounded language.
func RenderAdminLoginWithLocaleAndAppearance(w io.Writer, csrfToken, returnTo, locale string, appearance site.Appearance) error {
	if locale != "tr" {
		locale = "en"
	}
	var mfaChallenge bytes.Buffer
	if err := designsystem.AuthenticatorChallenge().Render(context.Background(), &mfaChallenge); err != nil {
		return err
	}
	if appearance.Normalize() != nil {
		appearance = site.Appearance{LoginOverlayOpacity: .4}
	}
	view := adminLoginView{
		Locale:       locale,
		CSRF:         csrfToken,
		ReturnTo:     returnTo,
		DocsURL:      "/docs",
		EnglishURL:   adminLoginURL("en", returnTo),
		TurkishURL:   adminLoginURL("tr", returnTo),
		MFAChallenge: localizeAdminLoginMFA(locale, mfaChallenge.String()),
	}
	if locale == "tr" {
		view.DocsURL = "/docs?lang=tr"
	}
	// The login styles live in static/admin/css/login.css. A configured
	// background image is passed as a custom property on the hero pane, so the
	// page needs no inline <style> block.
	if appearance.LoginBackgroundURL != "" {
		opacity := strconv.FormatFloat(appearance.LoginOverlayOpacity, 'f', 2, 64)
		view.HeroStyle = fmt.Sprintf(`--login-background:linear-gradient(rgba(9,10,15,%s),rgba(9,10,15,%s)),url("%s")`, opacity, opacity, appearance.LoginBackgroundURL)
	}
	return adminLoginPage(view).Render(context.Background(), w)
}

func adminLoginURL(language, returnTo string) string {
	query := url.Values{"lang": {language}}
	if returnTo != "/admin" {
		query.Set("return_to", returnTo)
	}
	return "/login?" + query.Encode()
}

// AdminLoginCSP returns hash sources for the exact inline blocks rendered on
// the login page. The page itself has none (its scripts are same-origin
// files), so this is empty unless a caller injects an inline block.
func AdminLoginCSP(document []byte) string {
	return inlineCSPDirectives(document)
}

var adminLoginEnglishMFA = strings.NewReplacer(
	`İki adımlı doğrulama`, `Two-step verification`,
	`Authenticator uygulamanızdaki güncel kodu veya kurtarma kodlarından birini girin.`, `Enter the current code from your authenticator app or one of your recovery codes.`,
	`Doğrulama kodu`, `Verification code`,
	`Sonraki kodu kullan`, `Use next code`,
)

func localizeAdminLoginMFA(locale, markup string) string {
	if locale == "tr" {
		return markup
	}
	return adminLoginEnglishMFA.Replace(markup)
}
