package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func TestParseBlockFormBuildsStructuredBlocks(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("block_count=1&block_type_0=hero&block_title_0=Welcome&block_button_url_0=%2Fstart&block_bg_color_0=primary_tint&block_padding_y_0=spacious"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	blocks, err := parseBlockForm(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0].Type != "hero" {
		t.Fatalf("blocks = %+v", blocks)
	}
	var payload map[string]string
	if err := json.Unmarshal(blocks[0].Payload, &payload); err != nil || payload["bg_color"] != "primary_tint" || payload["padding_y"] != "spacious" {
		t.Fatalf("block presets did not survive form parsing: %s err=%v", blocks[0].Payload, err)
	}
}

func TestParseBlockFormAcceptsTypedJSONWidgets(t *testing.T) {
	form := url.Values{"block_count": {"1"}, "block_type_0": {"hotspots"}, "block_payload_0": {`{"image":"/room.webp","points":[{"x":20,"y":35,"title":"Window"}]}`}}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	blocks, err := parseBlockForm(r)
	if err != nil || len(blocks) != 1 || blocks[0].Type != "hotspots" {
		t.Fatalf("blocks=%+v err=%v", blocks, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(blocks[0].Payload, &payload); err != nil || payload["image"] != "/room.webp" {
		t.Fatalf("payload=%s err=%v", blocks[0].Payload, err)
	}
}

func TestBlockEditorOffersAddMoveAndAllTypedBlocks(t *testing.T) {
	html := renderBlockEditorHTML(t, nil)
	for _, marker := range []string{"data-block-add", "data-block-up", "data-block-down", "before_after", "stats", "contact_cta", "video", "masonry", "hotspots", "block_bg_color_0", "block_padding_y_0", "data-block-field=\"payload\""} {
		if !strings.Contains(html, marker) {
			t.Errorf("editor is missing %q", marker)
		}
	}
}

func TestBlockEditorRestoresStylePresetSelections(t *testing.T) {
	html := renderBlockEditorHTML(t, []pages.Block{{Type: "hero", Payload: json.RawMessage(`{"title":"Title","bg_color":"dark","padding_y":"compact"}`)}})
	for _, marker := range []string{`<option value="dark" selected>`, `<option value="compact" selected>`, `name="block_bg_color_0"`, `name="block_padding_y_0"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("style preset editor missing %q", marker)
		}
	}
}

func TestPageStatusMapsMissingDependenciesToUnavailable(t *testing.T) {
	if got := pageStatus(service.ErrDependencyRequired); got != http.StatusServiceUnavailable {
		t.Fatalf("write status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := pageReadStatus(service.ErrDependencyRequired); got != http.StatusServiceUnavailable {
		t.Fatalf("read status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := pageReadStatus(errors.New("not found")); got != http.StatusNotFound {
		t.Fatalf("not-found status = %d, want %d", got, http.StatusNotFound)
	}
}

func TestNewIDProducesUniqueNonEmptyIDs(t *testing.T) {
	seen := make(map[string]struct{}, 1024)
	for i := 0; i < 1024; i++ {
		id, err := newID()
		if err != nil || id == "" {
			t.Fatalf("newID() = %q, %v", id, err)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate generated ID %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestSortPagesByViewsHonorsDirectionAndStableTies(t *testing.T) {
	items := []pages.Page{{ID: "c"}, {ID: "a"}, {ID: "b"}}
	counts := map[string]int64{"a": 5, "b": 5, "c": 2}
	sortPagesByViews(items, counts, "-views")
	if items[0].ID != "a" || items[1].ID != "b" || items[2].ID != "c" {
		t.Fatalf("descending view sort = %#v", items)
	}
	sortPagesByViews(items, counts, "views")
	if items[0].ID != "c" || items[1].ID != "a" || items[2].ID != "b" {
		t.Fatalf("ascending view sort = %#v", items)
	}
	before := append([]pages.Page(nil), items...)
	sortPagesByViews(items, counts, "unsupported")
	for i := range before {
		if before[i].ID != items[i].ID {
			t.Fatalf("unsupported sort changed list: %#v", items)
		}
	}
}

func renderBlockEditorHTML(t *testing.T, blocks []pages.Block) string {
	t.Helper()
	html, err := renderString(context.Background(), pageBlockEditor(newBlockEditorView(blocks, "csrf")))
	if err != nil {
		t.Fatal(err)
	}
	return html
}
