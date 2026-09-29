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

func TestBlockStylePresetUsesAllowlistedDesignTokens(t *testing.T) {
	html, err := Blocks([]pages.Block{{Type: "hero", Payload: json.RawMessage(`{"title":"Welcome","bg_color":"dark","padding_y":"compact"}`)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`ds-block-frame--dark`, `ds-block-frame--compact`, `data-block="hero"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("styled block missing %q: %s", marker, html)
		}
	}
}

func TestFAQBlockFeedbackUsesOptionalRenderContract(t *testing.T) {
	payload := json.RawMessage(`{"items":[{"question":"When?","answer":"Tomorrow"}]}`)
	without, err := Blocks([]pages.Block{{Type: "faq", Payload: payload}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(without, "data-feedback-type") {
		t.Fatal("feedback rendered without analytics option")
	}
	with, err := BlocksWithOptions([]pages.Block{{Type: "faq", Payload: payload}}, RenderOptions{FeedbackEnabled: true, CSRF: "csrf", Nonce: "nonce", FeedbackCounts: func(ids []string) map[string][2]int64 { return map[string][2]int64{ids[0]: {2, 1}} }})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`data-feedback-type="faq"`, `hx-post="/api/v1/feedback"`, `2 kişi faydalı buldu · 1 kişi faydalı bulmadı`, `name="_csrf" value="csrf"`} {
		if !strings.Contains(with, marker) {
			t.Errorf("FAQ feedback output missing %q", marker)
		}
	}
}

func BenchmarkBlocks(b *testing.B) {
	blocks := []pages.Block{{Type: "hero", Payload: json.RawMessage(`{"title":"Welcome","subtitle":"A fast page"}`)}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Blocks(blocks); err != nil {
			b.Fatal(err)
		}
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

func TestBlocksSafelyFallbackForUnknownLegacyType(t *testing.T) {
	html, err := Blocks([]pages.Block{{Type: "custom", Payload: json.RawMessage(`{"title":"<script>x</script>"}`)}})
	if err != nil || !strings.Contains(html, "Some content is temporarily unavailable") || strings.Contains(html, "<script>") {
		t.Fatalf("unknown block fallback = %s, %v", html, err)
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

func TestCountdownBlockRendersValidatedTargetAndExpiredState(t *testing.T) {
	blocks := []pages.Block{{Type: "countdown", Payload: json.RawMessage(`{"target_datetime":"2027-01-01T12:00:00+03:00","timezone":"Europe/Istanbul","expired_headline":"Registration is open","expired_cta_url":"/register"}`)}}
	html, err := Blocks(blocks)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"data-countdown", "2027-01-01T12:00:00+03:00", "Registration is open", "/register", "data-countdown-seconds"} {
		if !strings.Contains(html, want) {
			t.Errorf("countdown missing %q: %s", want, html)
		}
	}
	blocks[0].Payload = json.RawMessage(`{"target_datetime":"not-a-date","expired_headline":"Expired"}`)
	if err := pages.ValidateBlocks(blocks); err == nil {
		t.Fatal("invalid countdown target accepted")
	}
}
