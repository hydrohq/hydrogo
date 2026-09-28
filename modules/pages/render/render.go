package render

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

var ErrInvalidBlock = errors.New("cannot render invalid block")

var componentBufferPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// Blocks returns HTML generated only from validated, allowlisted block data.
// The returned string is safe to pass to a trusted template component; it is
// never copied from a client-provided HTML field.
func Blocks(blocks []pages.Block) (string, error) {
	if err := pages.ValidateBlocksForRender(blocks); err != nil {
		return "", ErrInvalidBlock
	}
	var out strings.Builder
	for _, block := range blocks {
		html, err := blockHTML(block)
		if err != nil {
			return "", err
		}
		styled, err := StyleBlockHTML(block, html)
		if err != nil {
			return "", err
		}
		out.WriteString(styled)
	}
	return out.String(), nil
}

// StyleBlockHTML applies validated block presets around safe HTML produced by
// this renderer or by a trusted Client Edition theme template.
func StyleBlockHTML(block pages.Block, html string) (string, error) {
	var payload struct {
		Background string `json:"bg_color"`
		Padding    string `json:"padding_y"`
	}
	if err := json.Unmarshal(block.Payload, &payload); err != nil {
		return "", ErrInvalidBlock
	}
	if payload.Background == "" && payload.Padding == "" {
		return html, nil
	}
	ctx := templ.WithChildren(context.Background(), templ.Raw(html))
	return renderComponentWithContext(designsystem.BlockStyleFrame(designsystem.BlockStyleProps{Background: payload.Background, Padding: payload.Padding}), ctx)
}

func blockHTML(block pages.Block) (string, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(block.Payload, &data); err != nil {
		return "", ErrInvalidBlock
	}
	switch block.Type {
	case "countdown":
		var p designsystem.CountdownProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.Countdown(p))
	case "hero":
		var p struct {
			Title        string `json:"title"`
			Subtitle     string `json:"subtitle"`
			MediaAssetID string `json:"media_asset_id"`
			MediaURL     string `json:"media_url"`
			MediaAlt     string `json:"media_alt"`
			ButtonText   string `json:"button_text"`
			ButtonURL    string `json:"button_url"`
		}
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.HeroBlock(designsystem.HeroProps{Title: p.Title, Subtitle: p.Subtitle, MediaAssetID: p.MediaAssetID, MediaURL: p.MediaURL, MediaAlt: p.MediaAlt, ButtonText: p.ButtonText, ButtonURL: p.ButtonURL}))
	case "cta":
		var p designsystem.CTAProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.CTABlock(p))
	case "faq":
		var items []designsystem.FAQItem
		if err := json.Unmarshal(data["items"], &items); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.FAQList(designsystem.FAQProps{Heading: "Frequently asked questions", Items: items}))
	case "before_after":
		var p struct {
			BeforeID    string `json:"before_image_asset_id"`
			AfterID     string `json:"after_image_asset_id"`
			Before      string `json:"before_image_url"`
			After       string `json:"after_image_url"`
			BeforeAlt   string `json:"before_alt"`
			AfterAlt    string `json:"after_alt"`
			BeforeLabel string `json:"before_label"`
			AfterLabel  string `json:"after_label"`
			Position    int    `json:"initial_slider_position"`
		}
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		_, positionSet := data["initial_slider_position"]
		return renderComponent(designsystem.BeforeAfter(designsystem.BeforeAfterProps{BeforeSrc: p.Before, AfterSrc: p.After, BeforeAlt: p.BeforeAlt, AfterAlt: p.AfterAlt, BeforeLabel: p.BeforeLabel, AfterLabel: p.AfterLabel, Position: p.Position, PositionSet: positionSet}))
	case "stats":
		var payload struct {
			Stats []designsystem.CounterStat `json:"stats"`
		}
		if err := json.Unmarshal(block.Payload, &payload); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.AnimatedCounter(designsystem.CounterProps{Stats: payload.Stats}))
	case "contact_cta":
		var p designsystem.ContactCTAProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.ContactCTA(p))
	case "video":
		var p designsystem.VideoProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.VideoPlayer(p))
	case "masonry":
		var p designsystem.MasonryProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.MasonryGrid(p))
	case "hotspots":
		var p designsystem.HotspotsProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.ImageHotspots(p))
	case "pricing_table":
		var p designsystem.PricingTableProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.PricingTable(p))
	case "testimonials":
		var p designsystem.TestimonialsProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.Testimonials(p))
	case "callout":
		var p designsystem.CalloutProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.Callout(p))
	case "action_card":
		var p designsystem.ActionCardProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.ActionCard(p))
	case "carousel":
		var p designsystem.CarouselProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.ImageCarousel(p))
	case "columns":
		var p designsystem.ColumnsProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.ColumnLayout(p))
	case "feature_list":
		var p designsystem.FeatureListProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.FeatureList(p))
	case "video_hero":
		var p designsystem.VideoHeroProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.VideoHero(p))
	case "trust_bar":
		var p designsystem.TrustBarProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.TrustBar(p))
	case "figure":
		var p designsystem.FigureProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.SemanticFigure(p))
	case "business_status":
		var p designsystem.BusinessStatusProps
		if err := json.Unmarshal(block.Payload, &p); err != nil {
			return "", ErrInvalidBlock
		}
		return renderComponent(designsystem.BusinessStatusBadge(p))
	default:
		return `<aside class="ds-block-unavailable" role="status">Some content is temporarily unavailable.</aside>`, nil
	}
}

func renderComponent(component templ.Component) (string, error) {
	return renderComponentWithContext(component, context.Background())
}

func renderComponentWithContext(component templ.Component, ctx context.Context) (string, error) {
	out := componentBufferPool.Get().(*bytes.Buffer)
	out.Reset()
	defer func() {
		if out.Cap() <= 256<<10 {
			out.Reset()
			componentBufferPool.Put(out)
		}
	}()
	if err := component.Render(ctx, out); err != nil {
		return "", err
	}
	return string(out.Bytes()), nil
}
