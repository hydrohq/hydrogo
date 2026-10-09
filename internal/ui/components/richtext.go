package components

import "github.com/a-h/templ"

// RichTextEditorProps describes a server-rendered rich text control.
type RichTextEditorProps struct {
	ID, Name, Label, Value                                                      string
	ToolbarLabel, BoldLabel, ItalicLabel, UnderlineLabel, LinkLabel, ImageLabel string
	BlockQuoteLabel, OrderedListLabel, BulletListLabel                          string
}

func RichTextEditor(p RichTextEditorProps) templ.Component {
	setRichTextDefault := func(target *string, fallback string) {
		if *target == "" {
			*target = fallback
		}
	}
	setRichTextDefault(&p.ToolbarLabel, p.Label+" araçları")
	setRichTextDefault(&p.BoldLabel, "Kalın")
	setRichTextDefault(&p.ItalicLabel, "İtalik")
	setRichTextDefault(&p.UnderlineLabel, "Altı çizili")
	setRichTextDefault(&p.LinkLabel, "Bağlantı ekle")
	setRichTextDefault(&p.ImageLabel, "Görsel ekle")
	setRichTextDefault(&p.BlockQuoteLabel, "Alıntı")
	setRichTextDefault(&p.OrderedListLabel, "Numaralı liste")
	setRichTextDefault(&p.BulletListLabel, "Madde listesi")
	return richTextEditorTemplate(p)
}
