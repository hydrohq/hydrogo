package media

import (
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

func TestMediaLibraryViewEscapesAssetsAndLoadsStaticScripts(t *testing.T) {
	view := mediaLibraryView{CSRF: "csrf&token", Cards: []mediaCardView{
		{ID: "asset-1", Filename: `logo"<script>.png`, Meta: "10 B", Preview: "image", ObjectPosition: "25% 75%", Metadata: ui.MediaMetadataFormProps{Action: "/admin/media/asset-1/metadata"}},
		{ID: "asset-2", Filename: "clip.mp4", Private: true, Preview: "video"},
	}}
	var out strings.Builder
	if err := mediaLibrary(view).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, marker := range []string{
		`<script defer src="/admin-assets/js/media-library.js"></script>`,
		`<script defer src="/admin/media/static/uploader.js"></script>`,
		`id="media-dropzone"`, `id="media-file-input"`, `id="media-count-badge"`,
		`2 asset`, `csrf&amp;token`, `logo&#34;&lt;script&gt;.png`,
		`object-position:25% 75%`, `<span class="media-badge is-private">Özel</span>`,
		`action="/admin/media/asset-1/visibility"`, `data-copy="asset-1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("media library missing %q", marker)
		}
	}
	if strings.Contains(body, "<script>") || strings.Contains(body, "<style") {
		t.Fatal("media library must not carry inline scripts or styles")
	}
}

func TestMediaLibraryViewShowsEmptyState(t *testing.T) {
	var out strings.Builder
	if err := mediaLibrary(mediaLibraryView{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `class="empty media-empty"`) || !strings.Contains(out.String(), "0 asset") {
		t.Fatalf("empty state missing: %s", out.String())
	}
}
