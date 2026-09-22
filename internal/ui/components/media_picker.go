package components

import (
	"context"
	"io"

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
		return writeString(w, `<div class="media-picker"><label for="`+esc(id)+`">`+esc(p.Label)+`</label><input id="`+esc(id)+`" name="`+esc(name)+`" type="hidden" value="`+esc(p.Value)+`"><a class="button button-quiet" href="/admin/media?select=`+esc(id)+`">Medya seç</a></div>`)
	})
}

func writeString(w io.Writer, value string) error {
	_, err := io.WriteString(w, value)
	return err
}
