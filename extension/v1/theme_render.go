package v1

import (
	htmltemplate "html/template"
	"io"
	"text/template"
)

// RenderPage renders a loaded theme layout with named, already-sanitized
// slots. Slot names are validated by the manifest and unknown slots resolve
// to empty output.
func (b ThemeBundle) RenderPage(w io.Writer, data PageTemplateData) error {
	if w == nil || data.Slots == nil {
		return ErrInvalidThemeManifest
	}
	tpl, err := template.New(b.Manifest.Name).Funcs(template.FuncMap{
		"slot": func(name string) htmltemplate.HTML { return data.Slots[name] },
	}).Parse(b.Layout)
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}
