package pages

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateBlocksAllowsSupportedPayloads(t *testing.T) {
	blocks := []Block{
		{Type: "hero", Payload: json.RawMessage(`{"title":"Welcome","button_url":"/start","media_asset_id":"asset-1"}`)},
		{Type: "faq", Payload: json.RawMessage(`{"items":[{"question":"Q","answer":"A"}]}`)},
		{Type: "pricing_table", Payload: json.RawMessage(`{"items":[{"name":"Pro","price_cents":1299,"currency":"TRY","features":["Email support"]}]}`)},
		{Type: "testimonials", Payload: json.RawMessage(`{"items":[{"name":"Ada","quote":"Excellent"}]}`)},
	}
	if err := ValidateBlocks(blocks); err != nil {
		t.Fatalf("valid blocks rejected: %v", err)
	}
}

func TestValidateBlocksRejectsUnsafeOrUnknownPayloads(t *testing.T) {
	cases := []Block{
		{Type: "unknown", Payload: json.RawMessage(`{"title":"x"}`)},
		{Type: "hero", Payload: json.RawMessage(`{"title":"x","button_url":"javascript:alert(1)"}`)},
		{Type: "hero", Payload: json.RawMessage(`{"title":"x","media_asset_id":"asset id"}`)},
		{Type: "hero", Payload: json.RawMessage(`{"button_url":"/missing-title"}`)},
	}
	for _, block := range cases {
		if !errors.Is(ValidateBlocks([]Block{block}), ErrInvalidBlocks) {
			t.Fatalf("block should be rejected: %+v", block)
		}
	}
}

func TestBlockStoreRoundTrip(t *testing.T) {
	want := []Block{{Type: "cta", Payload: json.RawMessage(`{"title":"Buy"}`)}}
	encoded, err := EncodeBlocksForStore(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeBlocksForStore(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Type != "cta" {
		t.Fatalf("round trip = %+v", got)
	}
}

func TestValidateDesignSystemBlocks(t *testing.T) {
	valid := []Block{
		{Type: "before_after", Payload: json.RawMessage(`{"before_image_asset_id":"before-1","after_image_asset_id":"after-1","initial_slider_position":50}`)},
		{Type: "stats", Payload: json.RawMessage(`{"stats":[{"number_value":2500,"suffix":"+","label":"Projects","icon_badge":"★"}]}`)},
		{Type: "contact_cta", Payload: json.RawMessage(`{"phone":"+1 415 555 0132","whatsapp":"14155550132","message":"Hello & welcome"}`)},
		{Type: "video", Payload: json.RawMessage(`{"src":"/media/video.mp4","type":"video/mp4","tracks":[{"src":"/media/captions.vtt","language":"en","label":"English"}]}`)},
		{Type: "masonry", Payload: json.RawMessage(`{"items":[{"image":"/media/a.webp","alt":"A room","caption":"Before","link":"/gallery/a"}]}`)},
		{Type: "hotspots", Payload: json.RawMessage(`{"image":"/media/room.webp","points":[{"x":20,"y":35,"title":"Window","description":"Triple glazed"}]}`)},
	}
	if err := ValidateBlocks(valid); err != nil {
		t.Fatalf("valid design blocks rejected: %v", err)
	}
	invalid := []Block{
		{Type: "before_after", Payload: json.RawMessage(`{"before_image_asset_id":"a","after_image_asset_id":"b","initial_slider_position":101}`)},
		{Type: "stats", Payload: json.RawMessage(`{"stats":[{"number_value":-1,"label":"Bad"}]}`)},
		{Type: "contact_cta", Payload: json.RawMessage(`{"phone":"javascript:alert(1)"}`)},
		{Type: "video", Payload: json.RawMessage(`{"src":"javascript:alert(1)"}`)},
		{Type: "masonry", Payload: json.RawMessage(`{"items":[{"image":"/x.webp","alt":"x","link":"javascript:alert(1)"}]}`)},
		{Type: "hotspots", Payload: json.RawMessage(`{"image":"/x.webp","points":[{"x":101,"y":20,"title":"x"}]}`)},
	}
	for _, block := range invalid {
		if err := ValidateBlocks([]Block{block}); err == nil {
			t.Errorf("invalid %s block accepted", block.Type)
		}
	}
}

func TestUnknownStoredBlockIsDisplayableButCannotBeWritten(t *testing.T) {
	unknown := []Block{{Type: "edition_custom", Payload: json.RawMessage(`{"private":"value"}`)}}
	if err := ValidateBlocks(unknown); err == nil {
		t.Fatal("new unknown block write was accepted")
	}
	encoded := `[ {"type":"edition_custom","payload":{"private":"value"}} ]`
	decoded, err := DecodeBlocksForStore(encoded)
	if err != nil || len(decoded) != 1 || decoded[0].Type != "edition_custom" {
		t.Fatalf("legacy block read = %+v, %v", decoded, err)
	}
	if err := ValidateBlocksForRender([]Block{{Type: "edition_custom", Payload: json.RawMessage(`not-json`)}}); err == nil {
		t.Fatal("malformed legacy block was accepted")
	}
}
