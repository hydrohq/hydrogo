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
	Nonce         string
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
		if previewURL != "" {
			previewTarget := strings.TrimSpace(p.PreviewTarget)
			if previewTarget == "" {
				previewTarget = "#" + id + "-preview"
			}
			previewAttrs = ` hx-post="` + esc(previewURL) + `" hx-target="` + esc(previewTarget) + `" hx-swap="innerHTML" hx-trigger="keyup delay:300ms"`
		}
		if _, err := io.WriteString(w, `<fieldset class="block-editor" data-hydro-block-editor="`+esc(id)+`"><legend>`+esc(label)+`</legend><div class="block-editor-layout"><div class="block-editor-canvas"><div id="`+esc(id)+`-mount" data-editorjs-mount="`+esc(id)+`"></div><textarea id="`+esc(id)+`" name="`+esc(name)+`" hidden`+previewAttrs+`>`+esc(p.Value)+`</textarea></div><aside id="`+esc(id)+`-preview" class="block-editor-preview" aria-live="polite"><span>Önizleme bekleniyor.</span></aside></div><script nonce="`+esc(p.Nonce)+`" src="`+esc(script)+`"></script><script nonce="`+esc(p.Nonce)+`">window.HydroInitEditorJS=window.HydroInitEditorJS||function(config){if(!window.EditorJS)return;const input=document.getElementById(config.input);if(!input)return;let data={};try{data=JSON.parse(input.value||"{}")}catch(_e){}const editor=new window.EditorJS({holder:config.mount,data:data,onChange:async function(){try{input.value=JSON.stringify(await editor.save());input.dispatchEvent(new Event("keyup",{bubbles:true}));}catch(_e){}}});};window.HydroInitEditorJS({mount:"`+esc(id)+`-mount",input:"`+esc(id)+`"});</script></fieldset>`); err != nil {
			return err
		}
		return nil
	})
}
