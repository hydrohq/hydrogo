package adminstudio

import (
	"html/template"
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
	terms := v1.StableTaxonomyTerms(opts.Terms)
	selected := make(map[string]struct{}, len(opts.Selected))
	for _, id := range opts.Selected {
		selected[id] = struct{}{}
	}
	_, err := io.WriteString(w, `<fieldset class="studio-taxonomy-editor"><legend>İçerik sınıflandırması</legend><div class="studio-taxonomy-terms">`)
	if err != nil {
		return err
	}
	for _, term := range terms {
		id := strings.TrimSpace(term.ID)
		if id == "" {
			continue
		}
		_, err = io.WriteString(w, `<label class="studio-taxonomy-term"><input type="checkbox" name="taxonomy_term" value="`+template.HTMLEscapeString(id)+`"`)
		if _, ok := selected[id]; ok {
			_, err = io.WriteString(w, ` checked`)
		}
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, `><span>`+template.HTMLEscapeString(termLabel(term))+`</span><small>`+template.HTMLEscapeString(string(term.Kind))+`</small></label>`)
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, `</div></fieldset>`)
	return err
}

func termLabel(term v1.TaxonomyTerm) string {
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
