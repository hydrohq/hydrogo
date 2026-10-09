package components

import (
	"context"
	"io"
	"net/url"
	"strings"

	"github.com/a-h/templ"
)

// MediaPickerProps describes a typed reference to an already-authorized media
// item. The actual picker route and tenant policy remain Edition-owned.
type MediaPickerProps struct {
	ID    string
	Name  string
	Label string
	Value string
}

// MediaPicker renders a safe hidden media reference and a non-authorizing
// chooser link. The link is intentionally just a UI affordance; the mutation
// handler must validate ownership and permissions again.
func MediaPicker(p MediaPickerProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		id := p.ID
		if id == "" {
			id = "media"
		}
		name := p.Name
		if name == "" {
			name = id
		}
		selected := strings.TrimSpace(p.Value) != ""
		selectedClass, imageClass, imageURL, valueLabel, clearClass := "", "is-hidden", "", "Görsel seçilmedi", " is-hidden"
		if selected {
			selectedClass, imageClass, imageURL, valueLabel, clearClass = " has-selected", "", "/media/"+url.PathEscape(p.Value), p.Value, ""
		}
		markup := `<div id="` + esc(id) + `-field" class="media-select-field` + selectedClass + `"><label class="studio-form-label" for="` + esc(id) + `">` + esc(p.Label) + `</label><div class="media-select-card` + selectedClass + `"><div class="media-select-preview"><img class="` + imageClass + `" src="` + esc(imageURL) + `" alt="Seçilen görsel"><div class="media-select-empty"><span>Görsel seçilmedi</span></div></div><div class="media-select-body"><div class="media-select-info"><span class="media-select-name">` + esc(valueLabel) + `</span></div><div class="media-select-actions"><button class="button button-quiet media-picker-open" type="button" data-media-target="` + esc(id) + `">Kütüphaneden seç</button><button class="button button-quiet button-danger media-select-clear` + clearClass + `" type="button">Kaldır</button><a class="text-link" href="/admin/media?select=` + url.QueryEscape(id) + `">Medya kütüphanesini aç</a></div></div></div><input type="hidden" id="` + esc(id) + `" name="` + esc(name) + `" value="` + esc(p.Value) + `"></div>`
		return writeString(w, markup)
	})
}

func writeString(w io.Writer, value string) error {
	_, err := io.WriteString(w, value)
	return err
}
