package main

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
)

const adminLocaleCookieName = "hydro_admin_locale"

func adminLocaleFromRequest(r *http.Request) string {
	if r != nil {
		if cookie, err := r.Cookie(adminLocaleCookieName); err == nil && i18n.IsSupportedCatalogLocale(cookie.Value) {
			return i18n.ResolveCatalogLocale(cookie.Value)
		}
	}
	return i18n.ResolveCatalogLocale("")
}

func setAdminLocaleCookie(w http.ResponseWriter, locale string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: adminLocaleCookieName, Value: i18n.ResolveCatalogLocale(locale), Path: "/admin", MaxAge: int((365 * 24 * time.Hour).Seconds()), Secure: secure, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}
