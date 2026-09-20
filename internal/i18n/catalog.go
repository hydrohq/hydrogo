package i18n

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

//go:embed catalogs/*.json
var catalogFS embed.FS

var catalogs = loadCatalogs()

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
		if message, ok := catalogs[candidate]; ok {
			if value, exists := message[key]; exists {
				return value
			}
		}
	}
	return key
}

// SupportedCatalogLocales returns a stable copy for admin selectors and tests.
func SupportedCatalogLocales() []string { return []string{"tr", "en"} }
