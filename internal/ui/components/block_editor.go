package components

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// BlockEditorProps describes the provider-neutral publishing editor. The
// Editor.js bundle is an Edition asset, not a Core dependency.
type BlockEditorProps struct {
	ID            string
	Name          string
	Label         string
	Value         string
	EditorScript  string
	PreviewURL    string
	PreviewTarget string
}

// BlockEditor renders a JSON-backed Editor.js mount point and a server-side
// HTMX preview target. The hidden textarea remains the canonical form value.
func BlockEditor(p BlockEditorProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = "hydro-block-editor"
		}
		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = id
		}
		label := p.Label
		if strings.TrimSpace(label) == "" {
			label = "İçerik"
		}
		script := strings.TrimSpace(p.EditorScript)
		if script == "" {
			script = "/vendor/editorjs/editorjs.umd.js"
		}
		previewURL := strings.TrimSpace(p.PreviewURL)
		previewAttrs := ""
		previewMarker := ""
		if previewURL != "" {
			previewTarget := strings.TrimSpace(p.PreviewTarget)
			if previewTarget == "" {
				previewTarget = "#" + id + "-preview"
			}
			previewAttrs = ` hx-post="` + esc(previewURL) + `" hx-target="` + esc(previewTarget) + `" hx-swap="innerHTML" hx-trigger="keyup delay:300ms" hx-include="closest form" hx-params="body,_csrf"`
			previewMarker = ` data-editorjs-preview="true"`
		}
		if _, err := io.WriteString(w, `<fieldset class="block-editor" data-hydro-block-editor="`+esc(id)+`"><legend>`+esc(label)+`</legend><div class="block-editor-layout"><div class="block-editor-canvas"><div id="`+esc(id)+`-mount" data-editorjs-mount="`+esc(id)+`" data-editorjs-label="`+esc(label)+`" data-editorjs-script="`+esc(script)+`"`+previewMarker+`></div><textarea id="`+esc(id)+`" name="`+esc(name)+`" hidden`+previewAttrs+`>`+esc(p.Value)+`</textarea></div><aside id="`+esc(id)+`-preview" class="block-editor-preview" aria-live="polite"><span>Önizleme bekleniyor.</span></aside></div></fieldset>`); err != nil {
			return err
		}
		return nil
	})
}
