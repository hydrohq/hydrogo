package transport

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) navigationEditor(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	locales := supportedNavigationLocales(h.NavigationLocales)
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if !containsString(locales, locale) {
		locale = locales[0]
	}
	items, err := h.Service.ListAdminNavigation(r.Context(), locale)
	if err != nil {
		http.Error(w, "Menü bilgileri şu anda açılamıyor.", http.StatusServiceUnavailable)
		return
	}
	var suggestions []site.NavigationSuggestion
	if h.NavigationSuggestions != nil {
		suggestions, _ = h.NavigationSuggestions(r.Context(), locale)
	}
	view := navigationEditorView{Locale: locale, Locales: locales, CSRF: security.CSRFTokenFromContext(r.Context()), Suggestions: suggestions}
	for _, item := range items {
		view.Rows = append(view.Rows, navigationRowView{Key: item.ID, Item: item, Position: strconv.Itoa(item.Position), Parents: navigationParentOptions(item.ID, items)})
	}
	view.Template = navigationRowView{Key: navigationNewRowKey, Item: site.NavigationItem{Target: "_self", Visible: true}, Parents: navigationParentOptions("", nil)}
	if ui.AdminStartForRequest(w, r, "Menü düzenleyici", "Bağlantıları sıralayın ve alt menüler oluşturun", "site") != nil {
		return
	}
	_ = navigationEditorPage(view).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

// navigationParentOptions lists every other item as a possible parent.
func navigationParentOptions(id string, items []site.NavigationItem) []uicomponents.SelectOption {
	options := make([]uicomponents.SelectOption, 0, len(items)+1)
	options = append(options, uicomponents.SelectOption{Value: "", Label: "Ana menü"})
	for _, parent := range items {
		if parent.ID != id {
			options = append(options, uicomponents.SelectOption{Value: parent.ID, Label: parent.Label})
		}
	}
	return options
}

func supportedNavigationLocales(configured []string) []string {
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

func (h Handler) saveNavigationEditor(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Menü kaydedilemedi. Alanları kontrol edip tekrar deneyin.", http.StatusBadRequest)
		return
	}
	locale := canonical.Locale(r.FormValue("locale"))
	if !containsString(supportedNavigationLocales(h.NavigationLocales), locale) {
		http.Error(w, "Menü dili geçersiz.", http.StatusBadRequest)
		return
	}
	ids, labels, hrefs := r.Form["id"], r.Form["label"], r.Form["href"]
	parents, targets, visibility := r.Form["parent_id"], r.Form["target"], r.Form["visible"]
	if len(ids) > 500 || len(ids) != len(labels) || len(ids) != len(hrefs) || len(ids) != len(parents) || len(ids) != len(targets) || len(ids) != len(visibility) {
		http.Error(w, "Menüdeki bağlantı sayısı veya alanlar geçersiz.", http.StatusBadRequest)
		return
	}
	items := make([]site.NavigationItem, len(ids))
	for i := range ids {
		items[i] = site.NavigationItem{ID: ids[i], Locale: locale, Label: labels[i], Href: hrefs[i], ParentID: parents[i], Target: targets[i], Position: i, Visible: visibility[i] == "true"}
		if visibility[i] != "true" && visibility[i] != "false" {
			http.Error(w, "Menüdeki görünürlük seçimi geçersiz.", http.StatusBadRequest)
			return
		}
	}
	if err := h.Service.SaveNavigation(r.Context(), locale, items); err != nil {
		http.Error(w, "Menü kaydedilemedi. Bağlantıların ve üst menü seçimlerinin doğru olduğundan emin olun.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/navigation/editor?locale="+url.QueryEscape(locale)+"&saved=1", http.StatusSeeOther)
}
