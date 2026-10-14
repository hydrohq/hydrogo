package transport

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

// blockEditorView is the view model of the form-based block editor.
type blockEditorView struct {
	Rows []blockEditorRow
	CSRF string
	// Full disables the add button once pages.MaxBlocks rows exist.
	Full bool
}

// blockEditorRow holds the editable values of one block.
type blockEditorRow struct {
	Index                                                int
	Type, TypeLabel                                      string
	Title, Subtitle, ButtonText, ButtonURL, MediaAssetID string
	Items, Payload, Background, Padding                  string
}

type blockTypeOption struct{ Type, Label string }

// blockTypeOptions lists the selectable block types with their editor labels.
var blockTypeOptions = []blockTypeOption{{"hero", "Hero (Giriş Başlığı & Buton)"}, {"faq", "FAQ (Sıkça Sorulan Sorular)"}, {"cta", "CTA (Eyleme Çağrı)"}, {"pricing_table", "Fiyatlandırma Tablosu"}, {"testimonials", "Müşteri Yorumları"}, {"before_after", "Öncesi / Sonrası"}, {"stats", "İstatistikler"}, {"contact_cta", "Arama / WhatsApp İletişim"}, {"video", "Video"}, {"masonry", "Fotoğraf Galerisi (Masonry)"}, {"hotspots", "Görsel Vurguları (Hotspots)"}, {"callout", "Vurgu Kutusu (Callout)"}, {"action_card", "Hızlı Eylem Kartı"}, {"carousel", "Görsel Kaydırıcı (Carousel)"}, {"columns", "Sütunlar"}, {"feature_list", "Özellik Listesi"}, {"video_hero", "Video Hero"}, {"trust_bar", "Güven Rozetleri"}, {"figure", "Başlıklı Görsel"}, {"business_status", "Çalışma Durumu"}}

// blockTypeBadges are the short labels shown in each row header.
var blockTypeBadges = map[string]string{"hero": "Hero", "faq": "FAQ", "cta": "CTA", "pricing_table": "Pricing table", "testimonials": "Testimonials", "before_after": "Before / after", "stats": "Statistics", "contact_cta": "Call / WhatsApp", "video": "Video", "masonry": "Masonry gallery", "hotspots": "Image hotspots", "callout": "Callout", "action_card": "Quick action card", "carousel": "Image carousel", "columns": "Columns", "feature_list": "Feature list", "video_hero": "Video hero", "trust_bar": "Trust badges", "figure": "Image with caption", "business_status": "Business status"}

// newBlockEditorView builds the editor rows; an empty page gets one blank row.
func newBlockEditorView(blocks []pages.Block, csrf string) blockEditorView {
	count := len(blocks)
	if count == 0 {
		count = 1
	}
	if count > pages.MaxBlocks {
		count = pages.MaxBlocks
	}
	view := blockEditorView{Rows: make([]blockEditorRow, 0, count), CSRF: csrf, Full: count >= pages.MaxBlocks}
	for i := 0; i < count; i++ {
		row := blockEditorRow{Index: i, Items: "[]", Payload: "{}"}
		if i < len(blocks) {
			row.Type, row.Title, row.Subtitle, row.ButtonText, row.ButtonURL, row.MediaAssetID, row.Items = blockEditorValues(blocks[i])
			row.Background, row.Padding = blockStyleEditorValues(blocks[i])
			row.Payload = string(blocks[i].Payload)
		}
		row.TypeLabel = row.Type
		if badge, ok := blockTypeBadges[row.Type]; ok {
			row.TypeLabel = badge
		}
		if row.TypeLabel == "" {
			row.TypeLabel = "Yeni Blok"
		}
		view.Rows = append(view.Rows, row)
	}
	return view
}

func blockFieldName(prefix string, index int) string {
	return prefix + strconv.Itoa(index)
}

func blockEditorValues(block pages.Block) (string, string, string, string, string, string, string) {
	var data map[string]json.RawMessage
	if json.Unmarshal(block.Payload, &data) != nil {
		return block.Type, "", "", "", "", "", "[]"
	}
	get := func(key string) string { var value string; _ = json.Unmarshal(data[key], &value); return value }
	items := "[]"
	if raw, ok := data["items"]; ok {
		items = string(raw)
	}
	return block.Type, get("title"), get("subtitle"), get("button_text"), get("button_url"), get("media_asset_id"), items
}

func blockStyleEditorValues(block pages.Block) (string, string) {
	var data map[string]json.RawMessage
	if json.Unmarshal(block.Payload, &data) != nil {
		return "", ""
	}
	var background, padding string
	_ = json.Unmarshal(data["bg_color"], &background)
	_ = json.Unmarshal(data["padding_y"], &padding)
	return background, padding
}

func parseBlockForm(r *http.Request) ([]pages.Block, error) {
	count, err := strconv.Atoi(r.FormValue("block_count"))
	if err != nil || count < 0 || count > pages.MaxBlocks {
		return nil, pages.ErrInvalidBlocks
	}
	blocks := make([]pages.Block, 0, count)
	for i := 0; i < count; i++ {
		typ := strings.TrimSpace(r.FormValue("block_type_" + strconv.Itoa(i)))
		if typ == "" {
			continue
		}
		payload := map[string]any{"title": r.FormValue("block_title_" + strconv.Itoa(i))}
		if mediaAssetID := strings.TrimSpace(r.FormValue("block_media_asset_id_" + strconv.Itoa(i))); mediaAssetID != "" {
			payload["media_asset_id"] = mediaAssetID
		}
		if typ == "faq" {
			var items []map[string]string
			if json.Unmarshal([]byte(r.FormValue("block_items_"+strconv.Itoa(i))), &items) != nil {
				return nil, pages.ErrInvalidBlocks
			}
			payload = map[string]any{"items": items}
		} else if typ == "pricing_table" || typ == "testimonials" {
			var items []json.RawMessage
			if json.Unmarshal([]byte(r.FormValue("block_items_"+strconv.Itoa(i))), &items) != nil {
				return nil, pages.ErrInvalidBlocks
			}
			payload = map[string]any{"items": items}
		} else if isJSONPayloadBlock(typ) {
			payload = nil
			if json.Unmarshal([]byte(r.FormValue("block_payload_"+strconv.Itoa(i))), &payload) != nil || payload == nil {
				return nil, pages.ErrInvalidBlocks
			}
		} else {
			payload["subtitle"] = r.FormValue("block_subtitle_" + strconv.Itoa(i))
			payload["button_text"] = r.FormValue("block_button_text_" + strconv.Itoa(i))
			// An empty URL is "no button", not an unsafe link.
			if buttonURL := strings.TrimSpace(r.FormValue("block_button_url_" + strconv.Itoa(i))); buttonURL != "" {
				payload["button_url"] = buttonURL
			}
		}
		if background := r.FormValue("block_bg_color_" + strconv.Itoa(i)); background != "" && background != "default" {
			payload["bg_color"] = background
		}
		if padding := r.FormValue("block_padding_y_" + strconv.Itoa(i)); padding != "" && padding != "normal" {
			payload["padding_y"] = padding
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, pages.Block{Type: typ, Payload: raw})
	}
	if err := pages.ValidateBlocks(blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

func isJSONPayloadBlock(typ string) bool {
	switch typ {
	case "before_after", "stats", "contact_cta", "video", "masonry", "hotspots", "callout", "action_card", "carousel", "columns", "feature_list", "video_hero", "trust_bar", "figure", "business_status":
		return true
	}
	return false
}
