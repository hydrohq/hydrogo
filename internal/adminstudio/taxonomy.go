package adminstudio

import (
	"context"
	"io"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
)

// TaxonomyEditorOptions describes the presentation boundary only. The owning
// module remains responsible for loading terms, authorization and persistence.
type TaxonomyEditorOptions struct {
	Terms    []v1.TaxonomyTerm
	Selected []string
}

func RenderTaxonomyEditor(w io.Writer, opts TaxonomyEditorOptions) error {
	selected := make(map[string]struct{}, len(opts.Selected))
	for _, id := range opts.Selected {
		selected[id] = struct{}{}
	}
	var views []taxonomyTermView
	for _, term := range v1.StableTaxonomyTerms(opts.Terms) {
		id := strings.TrimSpace(term.ID)
		if id == "" {
			continue
		}
		_, isSelected := selected[id]
		views = append(views, taxonomyTermView{ID: id, Label: termLabel(term), Kind: string(term.Kind), Selected: isSelected})
	}
	return taxonomyEditorView(views).Render(context.Background(), w)
}

// taxonomyTermView is one checkbox of the taxonomy editor.
type taxonomyTermView struct {
	ID, Label, Kind string
	Selected        bool
}

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
