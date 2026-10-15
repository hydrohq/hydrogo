package adminstudio

import (
	"sort"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

func termLabel(term v1.TaxonomyTerm) string {
	if tr, ok := term.Labels["tr"]; ok && tr != "" {
		return tr
	}
	locales := make([]string, 0, len(term.Labels))
	for locale := range term.Labels {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	if len(locales) > 0 {
		return term.Labels[locales[0]]
	}
	return term.Slug
}
