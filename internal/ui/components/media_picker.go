package components

import (
	"net/url"
	"strings"

	"github.com/a-h/templ"
)

// MediaPickerProps describes a typed reference to an already-authorized media
// item. The actual picker route and tenant policy remain Edition-owned.
type MediaPickerProps struct {
	ID, Name, Label, Value                         string
	SelectedImageAlt, EmptySelection               string
	ChooseFromLibrary, ClearSelection, OpenLibrary string
}

// MediaPicker renders a safe hidden media reference and a non-authorizing
// chooser link. The link is intentionally just a UI affordance; the mutation
// handler must validate ownership and permissions again.
func MediaPicker(p MediaPickerProps) templ.Component {
	defaults := []struct {
		target   *string
		fallback string
	}{
		{&p.ID, "media"},
		{&p.SelectedImageAlt, "Seçilen görsel"},
		{&p.EmptySelection, "Görsel seçilmedi"},
		{&p.ChooseFromLibrary, "Kütüphaneden seç"},
		{&p.ClearSelection, "Kaldır"},
		{&p.OpenLibrary, "Medya kütüphanesini aç"},
	}
	for _, d := range defaults {
		if *d.target == "" {
			*d.target = d.fallback
		}
	}
	if p.Name == "" {
		p.Name = p.ID
	}
	return mediaPickerTemplate(p, strings.TrimSpace(p.Value) != "")
}

// mediaPickerImageURL is the preview source for the selected asset.
func mediaPickerImageURL(p MediaPickerProps, selected bool) string {
	if !selected {
		return ""
	}
	return "/media/" + url.PathEscape(p.Value)
}

// mediaPickerLabel names the selection, or says nothing is selected.
func mediaPickerLabel(p MediaPickerProps, selected bool) string {
	if !selected {
		return p.EmptySelection
	}
	return p.Value
}

func mediaLibraryHref(id string) string {
	return "/admin/media?select=" + url.QueryEscape(id)
}
