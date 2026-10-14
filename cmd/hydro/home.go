package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func publicHomePath(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}
	return "/public/pages/" + url.PathEscape(locale) + "/demo/html"
}

func configuredLocale(locale string, supported []string) string {
	locale = strings.TrimSpace(locale)
	for _, candidate := range supported {
		if strings.EqualFold(locale, candidate) {
			return candidate
		}
	}
	return ""
}

func cookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func requestedLocale(r *http.Request, supported []string, defaultLocale string) (string, bool) {
	if selected := configuredLocale(r.URL.Query().Get("locale"), supported); selected != "" {
		return selected, true
	}
	if selected := configuredLocale(cookieValue(r, "hydro_locale"), supported); selected != "" {
		return selected, false
	}
	return i18n.ResolveAcceptLanguage(r.Header.Get("Accept-Language"), supported, defaultLocale), false
}

func localeRedirect(w http.ResponseWriter, r *http.Request, locale string) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Add("Vary", "Accept-Language")
	w.Header().Add("Vary", "Cookie")
	http.Redirect(w, r, "/"+url.PathEscape(locale)+"/", http.StatusFound)
}

// technicalHomeWithLocales is the technical home page with a locale picker
// when more than one locale is configured.
func technicalHomeWithLocales(locale string, locales []string, demoPath string, demoEnabled bool) string {
	if len(locales) < 2 {
		return technicalHomeHTML(locale, demoPath, demoEnabled)
	}
	picker := designsystem.LocalePicker(designsystem.LocalePickerProps{Locales: locales, Selected: configuredLocale(locale, locales)})
	if markup := renderTechnicalHome(locale, "", picker); markup != "" {
		return markup
	}
	return technicalHomeHTML(locale, demoPath, demoEnabled)
}

// technicalHomeHTML is intentionally small: the default root is a runtime
// status surface, not a marketing page. Product-specific frontends can still
// be configured through site.home_path or opened from the optional demo seed.
func technicalHomeHTML(locale, demoPath string, demoEnabled bool, nonce ...string) string {
	styleNonce := ""
	if len(nonce) > 0 {
		styleNonce = strings.TrimSpace(nonce[0])
	}
	return renderTechnicalHome(locale, styleNonce, templ.NopComponent)
}

func renderTechnicalHome(locale, styleNonce string, picker templ.Component) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}
	var output bytes.Buffer
	if err := technicalHome(locale, styleNonce, picker).Render(context.Background(), &output); err != nil {
		return ""
	}
	return output.String()
}

func styleNonceAttrs(nonce string) templ.Attributes {
	if nonce == "" {
		return templ.Attributes{}
	}
	return templ.Attributes{"nonce": nonce}
}

// writeTrustedHTML writes a machine-owned page (technical home, docs portal)
// and allows its inline styles and scripts by hash. Only a policy that is
// already set is extended; an empty one stays empty.
func writeTrustedHTML(w http.ResponseWriter, markup string) {
	if policy := w.Header().Get("Content-Security-Policy"); policy != "" {
		w.Header().Set("Content-Security-Policy", security.MergeCSP(policy, security.InlineCSPDirectives([]byte(markup))))
	}
	_, _ = io.WriteString(w, markup)
}
