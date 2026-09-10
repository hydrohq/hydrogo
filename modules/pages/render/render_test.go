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

func TestBlocksRejectUnknownType(t *testing.T) {
	_, err := Blocks([]pages.Block{{Type: "custom", Payload: json.RawMessage(`{"title":"x"}`)}})
	if err == nil {
		t.Fatal("unknown block rendered")
	}
}
