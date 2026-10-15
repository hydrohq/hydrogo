package transport

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type navigationResponse struct {
	Locale  string                `json:"locale"`
	Locales []string              `json:"locales"`
	Items   []site.NavigationItem `json:"items"`
}

func (h Handler) navigationLocales() []string {
	configured := h.NavigationLocales
	if len(configured) == 0 {
		configured = []string{"tr", "en"}
	}
	locales := make([]string, 0, len(configured))
	for _, raw := range configured {
		locale := canonical.Locale(raw)
		if locale != "" && !containsString(locales, locale) {
			locales = append(locales, locale)
		}
	}
	if len(locales) == 0 {
		return []string{"tr"}
	}
	return locales
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (h Handler) navigationLocale(w http.ResponseWriter, r *http.Request) (string, bool) {
	locale := canonical.Locale(r.PathValue("locale"))
	if !containsString(h.navigationLocales(), locale) {
		api.WriteError(w, r, http.StatusNotFound, "unknown navigation locale", nil)
		return "", false
	}
	return locale, true
}

func (h Handler) adminNavigation(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	locale, ok := h.navigationLocale(w, r)
	if !ok {
		return
	}
	items, err := h.Service.ListAdminNavigation(r.Context(), locale)
	if err != nil {
		writeSiteError(w, r, err, "navigation unavailable")
		return
	}
	if items == nil {
		items = []site.NavigationItem{}
	}
	api.WriteData(w, http.StatusOK, navigationResponse{Locale: locale, Locales: h.navigationLocales(), Items: items}, nil)
}

func (h Handler) adminSaveNavigation(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	locale, ok := h.navigationLocale(w, r)
	if !ok {
		return
	}
	var payload struct {
		Items []site.NavigationItem `json:"items"`
	}
	if !decode(w, r, &payload) {
		return
	}
	if payload.Items == nil {
		payload.Items = []site.NavigationItem{}
	}
	for i := range payload.Items {
		payload.Items[i].Position = i
	}
	if err := h.Service.SaveNavigation(r.Context(), locale, payload.Items); err != nil {
		writeSiteError(w, r, err, "invalid navigation")
		return
	}
	items, err := h.Service.ListAdminNavigation(r.Context(), locale)
	if err != nil {
		writeSiteError(w, r, err, "navigation unavailable")
		return
	}
	api.WriteData(w, http.StatusOK, navigationResponse{Locale: locale, Locales: h.navigationLocales(), Items: items}, nil)
}
