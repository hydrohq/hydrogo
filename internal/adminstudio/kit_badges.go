package adminstudio

import (
	"html/template"
	"io"
	"sort"
)

// RenderKitBadges renders a compact, escaped capability summary for admin
// collection headers. It intentionally exposes names only, never provider or
// credential details.
func RenderKitBadges(w io.Writer, kits []Kit) error {
	ordered := append([]Kit(nil), kits...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	const tpl = `<div class="studio-kit-badges" aria-label="Etkin yetenekler">{{range .}}<span class="studio-kit-badge">{{.}}</span>{{end}}</div>`
	return template.Must(template.New("kit-badges").Parse(tpl)).Execute(w, ordered)
}
