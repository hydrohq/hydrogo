package transport

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) translations(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || guards.IsNil(h.Store) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	scope, _ := tenant.FromContext(r.Context())
	localizedRows := make(map[string]map[string]i18n.TranslationRow)
	for _, cellLocale := range i18n.SupportedCatalogLocales() {
		localizedRows[cellLocale] = make(map[string]i18n.TranslationRow)
		for _, row := range i18n.CatalogRows(scope.TenantID, cellLocale) {
			localizedRows[cellLocale][row.Key] = row
		}
	}
	view := translationMatrixView{CSRF: security.CSRFTokenFromContext(r.Context())}
	for _, cellLocale := range i18n.SupportedCatalogLocales() {
		view.Locales = append(view.Locales, translationLocaleView{Code: cellLocale, Label: translationLocaleLabel(cellLocale)})
	}
	for _, row := range i18n.CatalogRows(scope.TenantID, "tr") {
		item := translationRowView{Key: row.Key, Default: row.Default}
		searchText := row.Key + " " + row.Default
		for _, cellLocale := range i18n.SupportedCatalogLocales() {
			localized := localizedRows[cellLocale][row.Key]
			searchText += " " + localized.Translation
			item.Cells = append(item.Cells, translationCellView{ID: "translation-" + cellLocale + "-" + strings.ReplaceAll(row.Key, ".", "-"), Key: row.Key, Locale: cellLocale, Value: localized.Translation, Missing: localized.Missing})
		}
		item.Search = strings.ToLower(searchText)
		view.Rows = append(view.Rows, item)
	}
	if ui.AdminStartForRequest(w, r, "Çeviri düzenleyici", "Sabit arayüz metinlerini düzenleyin", "site") != nil {
		return
	}
	_ = translationMatrix(view).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func translationLocaleLabel(locale string) string {
	switch locale {
	case "tr":
		return "Türkçe (TR)"
	case "en":
		return "English (EN)"
	}
	return strings.ToUpper(locale)
}

func (h Handler) saveTranslation(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid translation", http.StatusBadRequest)
		return
	}
	requestedLocale := strings.TrimSpace(r.FormValue("locale"))
	if !i18n.IsSupportedCatalogLocale(requestedLocale) {
		http.Error(w, "invalid translation locale", http.StatusBadRequest)
		return
	}
	locale := i18n.ResolveCatalogLocale(requestedLocale)
	key, value := strings.TrimSpace(r.FormValue("key")), strings.TrimSpace(r.FormValue("value"))
	if !i18n.IsCatalogKey(key) || len(value) > 1000 || strings.ContainsAny(value, "\x00\r\n") {
		http.Error(w, "invalid translation", http.StatusBadRequest)
		return
	}
	settingKey := i18n.OverrideSettingKey(locale, key)
	if err := h.Service.SaveSetting(r.Context(), site.Setting{Key: settingKey, Value: value, Public: true}); err != nil {
		http.Error(w, "translation could not be saved", siteStatus(err))
		return
	}
	scope, _ := tenant.FromContext(r.Context())
	if !i18n.SetOverride(scope.TenantID, locale, key, value) {
		http.Error(w, "translation could not be saved", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/translations?locale="+url.QueryEscape(locale)+"&saved=1", http.StatusSeeOther)
}
