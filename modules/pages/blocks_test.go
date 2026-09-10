package pages

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateBlocksAllowsSupportedPayloads(t *testing.T) {
	blocks := []Block{
		{Type: "hero", Payload: json.RawMessage(`{"title":"Welcome","button_url":"/start"}`)},
		{Type: "faq", Payload: json.RawMessage(`{"items":[{"question":"Q","answer":"A"}]}`)},
	}
	if err := ValidateBlocks(blocks); err != nil {
		t.Fatalf("valid blocks rejected: %v", err)
	}
}

func TestValidateBlocksRejectsUnsafeOrUnknownPayloads(t *testing.T) {
	cases := []Block{
		{Type: "unknown", Payload: json.RawMessage(`{"title":"x"}`)},
		{Type: "hero", Payload: json.RawMessage(`{"title":"x","button_url":"javascript:alert(1)"}`)},
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
