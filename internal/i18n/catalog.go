package i18n

import (
	"embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

//go:embed catalogs/*.json
var catalogFS embed.FS

var catalogs = loadCatalogs()

var overrides = struct {
	sync.RWMutex
	byTenant map[string]map[string]map[string]string
}{byTenant: make(map[string]map[string]map[string]string)}

type TranslationRow struct {
	Key         string
	Default     string
	Translation string
	Missing     bool
}

const settingPrefix = "i18n.override."

func OverrideSettingKey(locale, key string) string {
	return settingPrefix + ResolveCatalogLocale(locale) + "." + strings.TrimSpace(key)
}

func IsCatalogKey(key string) bool {
	_, ok := catalogs["tr"][strings.TrimSpace(key)]
	return ok
}

// OverrideSettings parses the dedicated Site setting namespace. Unknown
// locales, keys, and values are ignored so stale rows cannot affect rendering.
func OverrideSettings(settings map[string]string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for setting, value := range settings {
		if !strings.HasPrefix(setting, settingPrefix) {
			continue
		}
		name := strings.TrimPrefix(setting, settingPrefix)
		separator := strings.IndexByte(name, '.')
		if separator < 1 {
			continue
		}
		locale, key := name[:separator], name[separator+1:]
		if !IsSupportedCatalogLocale(locale) {
			continue
		}
		locale = ResolveCatalogLocale(locale)
		if !validCatalogOverride(locale, key, value) {
			continue
		}
		if result[locale] == nil {
			result[locale] = make(map[string]string)
		}
		result[locale][key] = strings.TrimSpace(value)
	}
	return result
}

func loadCatalogs() map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, locale := range []string{"tr", "en"} {
		data, err := catalogFS.ReadFile("catalogs/" + locale + ".json")
		if err != nil {
			continue
		}
		var messages map[string]string
		if json.Unmarshal(data, &messages) == nil {
			result[locale] = messages
		}
	}
	return result
}

// T returns a bounded local catalog translation. Missing keys intentionally
// return the key so untranslated text is visible to developers and never
// causes a runtime failure.
func T(locale, key string) string {
	return TForTenant("", locale, key)
}

// TForTenant applies editor overrides before falling back to the embedded
// catalog. Tenant identity is part of the lookup to keep editions isolated.
func TForTenant(tenantID, locale, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	requested := canonical.Locale(locale)
	chain, err := LocaleFallbackChain(requested, []string{"tr", "en"}, "tr")
	if err != nil {
		chain = []string{"tr"}
	}
	for _, candidate := range chain {
		overrides.RLock()
		value, exists := overrides.byTenant[tenantID][candidate][key]
		overrides.RUnlock()
		if exists && strings.TrimSpace(value) != "" {
			return value
		}
		if message, ok := catalogs[candidate]; ok {
			if value, exists := message[key]; exists {
				return value
			}
		}
	}
	return key
}

// CatalogRows returns the supported keys and locale values in stable key
// order, falling back to the primary Turkish catalog for missing translations.
func CatalogRows(tenantID, locale string) []TranslationRow {
	locale = ResolveCatalogLocale(locale)
	primary := catalogs["tr"]
	keys := make([]string, 0, len(primary))
	for key := range primary {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([]TranslationRow, 0, len(keys))
	overrides.RLock()
	defer overrides.RUnlock()
	for _, key := range keys {
		value, ok := overrides.byTenant[tenantID][locale][key]
		if !ok {
			value, ok = catalogs[locale][key]
		}
		missing := !ok || strings.TrimSpace(value) == ""
		if missing {
			value = primary[key]
		}
		rows = append(rows, TranslationRow{Key: key, Default: primary[key], Translation: value, Missing: missing})
	}
	return rows
}

// SetOverride updates one validated catalog key in the bounded process cache.
func SetOverride(tenantID, locale, key, value string) bool {
	locale = ResolveCatalogLocale(locale)
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if len(value) > 1000 || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	if _, ok := catalogs[locale][key]; !ok {
		return false
	}
	overrides.Lock()
	defer overrides.Unlock()
	if overrides.byTenant[tenantID] == nil {
		overrides.byTenant[tenantID] = make(map[string]map[string]string)
	}
	if overrides.byTenant[tenantID][locale] == nil {
		overrides.byTenant[tenantID][locale] = make(map[string]string)
	}
	if value == "" {
		delete(overrides.byTenant[tenantID][locale], key)
	} else {
		overrides.byTenant[tenantID][locale][key] = value
	}
	return true
}

// ReplaceOverrides atomically replaces one tenant's persisted translation
// overrides, allowing removed DB keys to disappear from the local cache.
func ReplaceOverrides(tenantID string, values map[string]map[string]string) {
	clean := make(map[string]map[string]string)
	for locale, entries := range values {
		locale = ResolveCatalogLocale(locale)
		for key, value := range entries {
			if !validCatalogOverride(locale, key, value) {
				continue
			}
			if clean[locale] == nil {
				clean[locale] = make(map[string]string)
			}
			clean[locale][key] = strings.TrimSpace(value)
		}
	}
	overrides.Lock()
	overrides.byTenant[tenantID] = clean
	overrides.Unlock()
}

func validCatalogOverride(locale, key, value string) bool {
	_, known := catalogs[locale][key]
	value = strings.TrimSpace(value)
	return known && value != "" && len(value) <= 1000 && !strings.ContainsAny(value, "\x00\r\n")
}

// SupportedCatalogLocales returns a stable copy for admin selectors and tests.
func SupportedCatalogLocales() []string { return []string{"tr", "en"} }

func ResolveCatalogLocale(requested string) string {
	chain, err := LocaleFallbackChain(requested, SupportedCatalogLocales(), "tr")
	if err == nil && len(chain) > 0 {
		return chain[0]
	}
	return "tr"
}

func IsSupportedCatalogLocale(requested string) bool {
	locale := canonical.Locale(requested)
	for _, supported := range SupportedCatalogLocales() {
		if locale == supported || strings.HasPrefix(locale, supported+"-") {
			return true
		}
	}
	return false
}
