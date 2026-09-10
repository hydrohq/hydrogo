package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestBlocksEscapesTextAndRendersKnownTypes(t *testing.T) {
	blocks := []pages.Block{{Type: "hero", Payload: json.RawMessage(`{"title":"<script>alert(1)</script>","subtitle":"Welcome"}`)}}
	html, err := Blocks(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<script>") || !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf("unsafe render: %s", html)
	}
}

func TestHeroRendersMediaAssetMetadataSafely(t *testing.T) {
	blocks := []pages.Block{{Type: "hero", Payload: json.RawMessage(`{"title":"Welcome","media_asset_id":"asset-1"}`)}}
	html, err := Blocks(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `data-media-asset-id="asset-1"`) {
		t.Fatalf("media asset metadata missing: %s", html)
	}
	if strings.Contains(html, `src=`) {
		t.Fatalf("renderer must not expose a provider URL: %s", html)
	}
}

func TestBlocksRejectUnknownType(t *testing.T) {
	_, err := Blocks([]pages.Block{{Type: "custom", Payload: json.RawMessage(`{"title":"x"}`)}})
	if err == nil {
		t.Fatal("unknown block rendered")
	}
}

func TestBlocksRenderPricingAndTestimonialsSafely(t *testing.T) {
	blocks := []pages.Block{
		{Type: "pricing_table", Payload: json.RawMessage(`{"items":[{"name":"Pro","price_cents":1299,"currency":"TRY","features":["<fast>"]}]}`)},
		{Type: "testimonials", Payload: json.RawMessage(`{"items":[{"name":"<Ada>","quote":"Great"}]}`)},
	}
	html, err := Blocks(blocks)
	if err != nil || !strings.Contains(html, "pricing_table") || !strings.Contains(html, "testimonials") || strings.Contains(html, "<fast>") {
		t.Fatalf("block render = %s, %v", html, err)
	}
}
