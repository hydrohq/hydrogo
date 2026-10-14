package adminstudio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func (h GeneratedAdminHandler) localeSelection(r *http.Request, collection Collection) (selectedLocale, error) {
	options, err := h.Options.Locales(r, collection)
	if err != nil || len(options.Allowed) == 0 || len(options.Allowed) > 32 || !localePattern.MatchString(options.Default) {
		return selectedLocale{}, errors.New("locale policy unavailable")
	}
	allowed := make(map[string]struct{}, len(options.Allowed))
	defaultFound := false
	for _, locale := range options.Allowed {
		if !localePattern.MatchString(locale) {
			return selectedLocale{}, errors.New("invalid Edition locale")
		}
		if _, exists := allowed[locale]; exists {
			return selectedLocale{}, errors.New("duplicate Edition locale")
		}
		allowed[locale] = struct{}{}
		defaultFound = defaultFound || locale == options.Default
	}
	if !defaultFound || len(r.URL.Query()["locale"]) > 1 {
		return selectedLocale{}, errors.New("invalid locale selection")
	}
	current := r.URL.Query().Get("locale")
	if current == "" {
		current = options.Default
	}
	if _, ok := allowed[current]; !ok {
		return selectedLocale{}, errors.New("locale is not available")
	}
	options.Allowed = append([]string(nil), options.Allowed...)
	return selectedLocale{Options: options, Current: current}, nil
}

func localeState(r *http.Request) selectedLocale {
	state, _ := r.Context().Value(localeContextKey{}).(selectedLocale)
	return state
}

func localizedTarget(target, locale string) string {
	if locale == "" {
		return target
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return target
	}
	query := parsed.Query()
	query.Set("locale", locale)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func localePath(path, locale string) string {
	return localizedTarget(path, locale)
}

func applyLocalizedValues(collection Collection, record, values map[string]any) error {
	encoded, err := json.Marshal(values)
	if err != nil || len(encoded) > generatedAdminBodyLimit {
		return errors.New("localized payload exceeds the resource limit")
	}
	allowed := make(map[string]struct{})
	for _, field := range collection.Fields {
		if field.Localized {
			allowed[field.ID] = struct{}{}
		}
	}
	if len(values) > len(allowed) {
		return errors.New("localized field count exceeds the resource contract")
	}
	for field, value := range values {
		if _, ok := allowed[field]; !ok {
			return errors.New("localized value is outside the resource contract")
		}
		record[field] = value
	}
	return nil
}

// generatedLocaleNav lists the collection's locales; the current one is
// marked and the others are GET forms that switch the locale.
func (h GeneratedAdminHandler) generatedLocaleNav(r *http.Request, collection Collection) generatedLocaleNav {
	if !collection.Localization {
		return generatedLocaleNav{}
	}
	state := localeState(r)
	nav := generatedLocaleNav{Enabled: true, Label: i18n.T(generatedAdminLocale(r), "admin.generated.locale.navigation")}
	for _, locale := range state.Options.Allowed {
		label := strings.ToUpper(locale)
		item := generatedLocaleNavItem{Label: label, Current: locale == state.Current}
		if !item.Current {
			item.Switch = getForm(r.URL.Path, []ui.HiddenInputProps{{Name: "locale", Value: locale}}, label)
		}
		nav.Items = append(nav.Items, item)
	}
	return nav
}

// generatedCopyLocaleActions offers copying the record from every other
// locale the user may copy from.
func (h GeneratedAdminHandler) generatedCopyLocaleActions(r *http.Request, collection Collection, id string, record map[string]any) *generatedActionGroup {
	locale := generatedAdminLocale(r)
	group := &generatedActionGroup{Title: i18n.T(locale, "admin.generated.locale.copy_heading")}
	expected := generatedRecordVersion(record)
	if collection.Localization {
		expected = recordVersion(record["locale_version"])
	}
	action := localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id)+"/locale-copy", localeState(r).Current)
	for _, from := range localeState(r).Options.Allowed {
		if from == localeState(r).Current || !h.Options.CanCopyLocale(r, collection, id, from) {
			continue
		}
		hidden := []ui.HiddenInputProps{{Name: "from_locale", Value: from}, {Name: "expected_version", Value: strconv.FormatInt(expected, 10)}}
		group.Forms = append(group.Forms, postForm(action, h.Options.CSRFToken(r), hidden, fmt.Sprintf(i18n.T(locale, "admin.generated.locale.copy_action"), strings.ToUpper(from)), ui.Quiet))
	}
	return group
}

func (h GeneratedAdminHandler) copyLocale(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if !collection.Localization || h.Options.CopyLocale == nil || h.Options.CanCopyLocale == nil {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm) != 3 || len(r.PostForm["from_locale"]) != 1 || len(r.PostForm["expected_version"]) != 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	from := r.PostFormValue("from_locale")
	currentLocale := localeState(r).Current
	allowed := false
	for _, locale := range localeState(r).Options.Allowed {
		allowed = allowed || locale == from
	}
	expected, err := strconv.ParseInt(r.PostFormValue("expected_version"), 10, 64)
	if err != nil || expected < 0 || from == currentLocale || !allowed || !h.Options.CanCopyLocale(r, collection, id, from) {
		http.Error(w, "locale copy unavailable", http.StatusForbidden)
		return
	}
	_, currentVersion, err := h.Options.GetLocalized(r, collection, id, currentLocale)
	if err != nil || currentVersion < 0 {
		h.apiError(w, r)
		return
	}
	if currentVersion != expected {
		http.Error(w, "document changed; reload before copying", http.StatusConflict)
		return
	}
	if err := h.Options.CopyLocale(r, collection, id, from, currentLocale, expected); err != nil {
		http.Error(w, "locale copy unavailable", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), currentLocale), http.StatusSeeOther)
}

func (h GeneratedAdminHandler) mutateLocale(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if !collection.Localization || localeState(r).Current == localeState(r).Options.Default {
		http.NotFound(w, r)
		return
	}
	if !collection.Drafts && !h.resourceAccess(r, collection, id).Update {
		http.Error(w, "Bu kaydı düzenleme yetkiniz yok.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	versions := r.PostForm["locale_version"]
	if len(versions) != 1 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	expected, err := strconv.ParseInt(versions[0], 10, 64)
	if err != nil || expected < 0 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	fields := make(map[string]any, len(collection.Fields))
	allowed := map[string]struct{}{"_csrf": {}, "locale_version": {}}
	for _, field := range collection.Fields {
		if field.ReadOnly {
			continue
		}
		allowed[field.ID] = struct{}{}
		values, exists := r.PostForm[field.ID]
		if !exists || !field.Localized {
			continue
		}
		value, err := parseAdminField(field, values)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		fields[field.ID] = value
	}
	for key, values := range r.PostForm {
		if _, ok := allowed[key]; !ok || len(values) != 1 {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
	}
	_, currentVersion, err := h.Options.GetLocalized(r, collection, id, localeState(r).Current)
	if err != nil || currentVersion < 0 {
		h.apiError(w, r)
		return
	}
	if currentVersion != expected {
		http.Error(w, "document changed; reload before saving", http.StatusConflict)
		return
	}
	if err := h.Options.SaveLocalized(r, collection, id, localeState(r).Current, fields, expected); err != nil {
		http.Error(w, "translation could not be saved", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), localeState(r).Current), http.StatusSeeOther)
}
