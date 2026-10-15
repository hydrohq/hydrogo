package transport

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

// blockType describes one block kind to the editor. PayloadMode tells the
// client which editing surface fits: "fields" (title, subtitle, button,
// media), "items" (a JSON list under "items") or "json" (free-form payload
// validated by pages.ValidateBlocks on save).
type blockType struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Badge       string `json:"badge"`
	PayloadMode string `json:"payload_mode"`
}

var blockTypes = []blockType{
	{"hero", "Hero", "Hero", "fields"},
	{"cta", "Call to action", "CTA", "fields"},
	{"faq", "FAQ", "FAQ", "items"},
	{"pricing_table", "Pricing table", "Pricing table", "items"},
	{"testimonials", "Testimonials", "Testimonials", "items"},
	{"before_after", "Before / after", "Before / after", "json"},
	{"stats", "Statistics", "Statistics", "json"},
	{"contact_cta", "Call / WhatsApp contact", "Call / WhatsApp", "json"},
	{"video", "Video", "Video", "json"},
	{"masonry", "Masonry gallery", "Masonry gallery", "json"},
	{"hotspots", "Image hotspots", "Image hotspots", "json"},
	{"callout", "Callout", "Callout", "json"},
	{"action_card", "Quick action card", "Quick action card", "json"},
	{"carousel", "Image carousel", "Image carousel", "json"},
	{"columns", "Columns", "Columns", "json"},
	{"feature_list", "Feature list", "Feature list", "json"},
	{"video_hero", "Video hero", "Video hero", "json"},
	{"trust_bar", "Trust badges", "Trust badges", "json"},
	{"figure", "Image with caption", "Image with caption", "json"},
	{"business_status", "Business status", "Business status", "json"},
}

// GET /api/v1/admin/pages/blocks describes the block catalog and the style
// presets every block accepts.
func (h Handler) adminBlockCatalog(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{
		"types":      blockTypes,
		"max_blocks": pages.MaxBlocks,
		"presets": map[string][]string{
			"bg_color":  {"default", "muted", "primary_tint", "dark"},
			"padding_y": {"normal", "compact", "spacious"},
		},
	}, nil)
}

type layoutOption struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Blocks      []pages.Block `json:"blocks"`
}

// GET /api/v1/admin/pages/layouts lists the layout templates a new page can
// start from (built-in plus configured).
func (h Handler) adminLayouts(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	layouts := append(pages.BuiltinLayoutTemplates(), h.LayoutTemplates...)
	if err := pages.ValidateLayoutTemplates(layouts); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "page layouts are unavailable", nil)
		return
	}
	options := make([]layoutOption, 0, len(layouts))
	for _, layout := range layouts {
		blocks := layout.Blocks
		if blocks == nil {
			blocks = []pages.Block{}
		}
		options = append(options, layoutOption{ID: layout.ID, Name: layout.Name, Description: layout.Description, Blocks: blocks})
	}
	api.WriteData(w, http.StatusOK, options, nil)
}
