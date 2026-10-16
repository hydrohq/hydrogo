package contenttypes

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// Localization of content types. A record stores every field in the default
// locale; values of localized fields in the other locales are collection
// translations with their own version. Status, search, revisions and list
// sorting use the record only, so a translation never changes whether a
// record is published, and restoring a revision leaves translations alone.

var errLocale = errors.New("content type locale is not available")

// Locales are the site's content locales. Localization is active when there
// is more than one (see Enabled); with one, localized fields behave like
// plain fields.
type Locales struct {
	Default string
	All     []string
}

// NewLocales normalizes the configured locales to lowercase. When i18n is
// disabled only the default locale is used.
func NewLocales(enabled bool, defaultLocale string, locales []string) Locales {
	result := Locales{Default: strings.ToLower(strings.TrimSpace(defaultLocale))}
	if result.Default != "" {
		result.All = append(result.All, result.Default)
	}
	if !enabled {
		return result
	}
	for _, locale := range locales {
		locale = strings.ToLower(strings.TrimSpace(locale))
		if locale != "" && !result.Has(locale) {
			result.All = append(result.All, locale)
		}
	}
	return result
}

// Enabled reports whether more than one locale is configured.
func (l Locales) Enabled() bool { return len(l.All) > 1 }

func (l Locales) Has(locale string) bool {
	for _, candidate := range l.All {
		if candidate == locale {
			return true
		}
	}
	return false
}

// translatable reports whether locale holds translations of a localized
// type: a configured locale other than the default.
func (a AdminAPI) translatable(typeID, locale string) bool {
	definition, ok := definitionByID(a.types(), typeID)
	return ok && definition.Localized() && a.Locales.Enabled() && locale != a.Locales.Default && a.Locales.Has(locale)
}

// localizedValues returns the record's values of the localized fields.
func localizedValues(definition Definition, data map[string]any) map[string]any {
	result := map[string]any{}
	for _, field := range definition.Fields {
		if value, ok := data[field.ID]; ok && field.Localized {
			result[field.ID] = value
		}
	}
	return result
}

// withFallback overlays the non-empty translated values on the record's.
func withFallback(base, translated map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(translated))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range translated {
		if value != nil && value != "" {
			result[key] = value
		}
	}
	return result
}

// LocaleOptions lists the site locales for the admin's locale tabs.
func (a AdminAPI) LocaleOptions(_ *http.Request, collection adminstudio.Collection) (adminstudio.LocaleOptions, error) {
	definition, ok := definitionByID(a.types(), collection.ID)
	if !ok || !definition.Localized() || !a.Locales.Enabled() {
		return adminstudio.LocaleOptions{}, errLocale
	}
	return adminstudio.LocaleOptions{Default: a.Locales.Default, Allowed: append([]string(nil), a.Locales.All...)}, nil
}

// GetLocalized returns a record's localized fields in locale and the version
// to send back when saving. For the default locale that is the record and
// its version; for another locale every localized field is present, empty
// where it has no translation yet, so the edit form shows what is missing.
func (a AdminAPI) GetLocalized(r *http.Request, collection adminstudio.Collection, id, locale string) (map[string]any, int64, error) {
	definition, ok := definitionByID(a.types(), collection.ID)
	if !ok || !definition.Localized() || !a.Locales.Has(locale) {
		return nil, 0, errLocale
	}
	if locale == a.Locales.Default {
		record, err := a.Service.GetRecord(r.Context(), collection.ID, id)
		if err != nil {
			return nil, 0, err
		}
		return localizedValues(definition, record.Data), int64(record.Version), nil
	}
	translation, err := a.Service.GetTranslation(r.Context(), collection.ID, id, locale)
	if err != nil {
		return nil, 0, err
	}
	values := make(map[string]any, len(definition.Fields))
	for _, field := range definition.Fields {
		if field.Localized {
			values[field.ID] = ""
			if value, ok := translation.Data[field.ID]; ok {
				values[field.ID] = value
			}
		}
	}
	return values, int64(translation.Version), nil
}

// SaveLocalized stores the localized fields of a non-default locale if the
// translation is still at expectedVersion. Empty values fall back to the
// default locale.
func (a AdminAPI) SaveLocalized(r *http.Request, collection adminstudio.Collection, id, locale string, fields map[string]any, expectedVersion int64) error {
	if !a.translatable(collection.ID, locale) {
		return errLocale
	}
	r = a.withActor(r)
	_, err := a.Service.SaveTranslation(r.Context(), collection.ID, id, locale, fields, int(expectedVersion))
	return err
}

// ListLocalized returns the stored translations of the listed records in
// locale. Untranslated fields are left out so the list shows the default
// locale's value.
func (a AdminAPI) ListLocalized(r *http.Request, collection adminstudio.Collection, ids []string, locale string) (map[string]map[string]any, error) {
	if !a.translatable(collection.ID, locale) {
		return nil, errLocale
	}
	translations, err := a.Service.ListTranslations(r.Context(), collection.ID, ids, locale)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]any, len(translations))
	for id, translation := range translations {
		result[id] = translation.Data
	}
	return result, nil
}

// CanCopyLocale offers copying another locale's values into the locale being
// edited. The default locale is never a copy target: its values are the
// record, edited and versioned through the normal form.
func (a AdminAPI) CanCopyLocale(r *http.Request, collection adminstudio.Collection, _ string, from string) bool {
	target := r.URL.Query().Get("locale")
	return a.translatable(collection.ID, target) && a.Locales.Has(from) && from != target
}

// CopyLocale saves the values readers of from currently see (its
// translation with the default locale as fallback) as the translation in
// to, if that is still at expectedVersion.
func (a AdminAPI) CopyLocale(r *http.Request, collection adminstudio.Collection, id, from, to string, expectedVersion int64) error {
	if !a.translatable(collection.ID, to) || !a.Locales.Has(from) || from == to {
		return errLocale
	}
	definition, _ := definitionByID(a.types(), collection.ID)
	record, err := a.Service.GetRecord(r.Context(), collection.ID, id)
	if err != nil {
		return err
	}
	values := localizedValues(definition, record.Data)
	if from != a.Locales.Default {
		source, err := a.Service.GetTranslation(r.Context(), collection.ID, id, from)
		if err != nil {
			return err
		}
		values = withFallback(values, source.Data)
	}
	r = a.withActor(r)
	_, err = a.Service.SaveTranslation(r.Context(), collection.ID, id, to, values, int(expectedVersion))
	return err
}

// publicLocale resolves the public API's ?locale= parameter: empty means
// the default locale; anything else must be a configured locale.
func (p PublicAPI) publicLocale(r *http.Request) (string, bool) {
	values := r.URL.Query()["locale"]
	if len(values) > 1 {
		return "", false
	}
	if len(values) == 0 || values[0] == "" {
		return p.Locales.Default, true
	}
	locale := strings.ToLower(values[0])
	return locale, p.Locales.Has(locale)
}

// translate overlays the records' translations in locale on their fields,
// field by field, keeping the default locale's value where a translation
// is missing or empty.
func (p PublicAPI) translate(r *http.Request, definition Definition, locale string, records []collections.Record) error {
	if !definition.Localized() || !p.Locales.Enabled() || locale == p.Locales.Default || len(records) == 0 {
		return nil
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	translations, err := p.Service.ListTranslations(r.Context(), definition.ID, ids, locale)
	if err != nil {
		return err
	}
	for index, record := range records {
		if translation, ok := translations[record.ID]; ok {
			records[index].Data = withFallback(record.Data, translation.Data)
		}
	}
	return nil
}
