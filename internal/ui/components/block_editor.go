package components

import (
	"strings"

	"github.com/a-h/templ"
)

// BlockEditorProps describes the provider-neutral publishing editor. The
// Editor.js bundle is an Edition asset, not a Core dependency.
type BlockEditorProps struct {
	ID             string
	Name           string
	Label          string
	Value          string
	EditorScript   string
	PreviewURL     string
	PreviewTarget  string
	PreviewPending string
}

// BlockEditor renders a JSON-backed Editor.js mount point and a server-side
// HTMX preview target. The hidden textarea remains the canonical form value.
func BlockEditor(p BlockEditorProps) templ.Component {
	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" {
		p.ID = "hydro-block-editor"
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = p.ID
	}
	if strings.TrimSpace(p.Label) == "" {
		p.Label = "İçerik"
	}
	if p.PreviewPending == "" {
		p.PreviewPending = "Önizleme bekleniyor."
	}
	p.EditorScript = strings.TrimSpace(p.EditorScript)
	if p.EditorScript == "" {
		p.EditorScript = "/vendor/editorjs/editorjs.umd.js"
	}
	p.PreviewURL = strings.TrimSpace(p.PreviewURL)
	p.PreviewTarget = strings.TrimSpace(p.PreviewTarget)
	if p.PreviewTarget == "" {
		p.PreviewTarget = "#" + p.ID + "-preview"
	}
	return blockEditorTemplate(p)
}

// blockEditorPreviewAttrs wires the live preview only when the Edition gave
// a preview route; the component never invents one.
func blockEditorPreviewAttrs(p BlockEditorProps) templ.Attributes {
	if p.PreviewURL == "" {
		return templ.Attributes{}
	}
	return templ.Attributes{
		"hx-post":    p.PreviewURL,
		"hx-target":  p.PreviewTarget,
		"hx-swap":    "innerHTML",
		"hx-trigger": "keyup delay:300ms",
		"hx-include": "closest form",
		"hx-params":  "body,_csrf",
	}
}
