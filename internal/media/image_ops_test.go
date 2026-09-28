package media

import (
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageHelpers(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			src.Set(x, y, color.RGBA{R: 220, A: 255})
		}
	}
	hash, err := BlurHash(src, 4, 3)
	if err != nil || len(hash) != 28 {
		t.Fatalf("BlurHash = %q, %v", hash, err)
	}
	colors := DominantColors(src, 3)
	if len(colors) == 0 || !strings.HasPrefix(colors[0], "#") {
		t.Fatalf("dominant colors = %v", colors)
	}
	cropped, err := CropImage(src, 5, 5, FocalPoint{X: 1, Y: .5})
	if err != nil || cropped.Bounds().Dx() != 5 || cropped.Bounds().Dy() != 5 {
		t.Fatalf("crop = %v, %v", cropped, err)
	}
	marked, err := ApplyWatermark(src, WatermarkOptions{Text: "© HydroGo", Position: "bottom-right", Opacity: .5})
	if err != nil || marked.Bounds() != src.Bounds() {
		t.Fatalf("watermark = %v, %v", marked, err)
	}
	if got := (FocalPoint{X: 1.2, Y: -.1}).ObjectPosition(); got != "100.00% 0.00%" {
		t.Fatalf("position = %q", got)
	}
}

func TestDocumentValidationAndHotlinkGuard(t *testing.T) {
	if ValidateDocument("cv.pdf", []byte("%PDF-1.7\n"), 100) != nil {
		t.Fatal("valid PDF rejected")
	}
	if ValidateDocument("cv.exe", []byte("%PDF-1.7"), 100) == nil || ValidateDocument("fake.pdf", []byte("not a PDF"), 100) == nil {
		t.Fatal("invalid document accepted")
	}
	guard := HotlinkGuard("https://example.test", []string{"partner.test"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, tc := range []struct {
		ref    string
		status int
	}{{"", 204}, {"https://example.test/page", 204}, {"https://sub.example.test/page", 403}, {"https://partner.test/x", 204}, {"https://www.google.com/search?q=asset", 204}, {"https://example.test.evil/x", 403}} {
		r := httptest.NewRequest(http.MethodGet, "/media/a", nil)
		if tc.ref != "" {
			r.Header.Set("Referer", tc.ref)
		}
		w := httptest.NewRecorder()
		guard.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("referer %q: status %d", tc.ref, w.Code)
		}
	}
}

func TestMissingReferences(t *testing.T) {
	got := MissingReferences(`<img src="/media/asset-valid-001"> {"media_asset_id":"asset-missing-1","cover_media_id":"asset-missing-2"} /media/asset-missing-1`, map[string]struct{}{"asset-valid-001": {}})
	if len(got) != 2 || got[0] != "asset-missing-1" || got[1] != "asset-missing-2" {
		t.Fatalf("missing references = %v", got)
	}
	all := MissingReferences(`{"before_image_asset_id":"asset-old-0001","after_image_asset_id":"asset-old-0002"}`, nil)
	if len(all) != 2 {
		t.Fatalf("block references = %v", all)
	}
}

func TestNormalizeFilenameAndOEmbed(t *testing.T) {
	if got := NormalizeFilename("Şirket Çalışması 2026.JPG"); got != "sirket-calismasi-2026.jpg" {
		t.Fatalf("normalized filename = %q", got)
	}
	if got, ok := OEmbedURL("https://youtu.be/abc_123"); !ok || got != "https://www.youtube-nocookie.com/embed/abc_123" {
		t.Fatalf("oembed = %q, %v", got, ok)
	}
	for _, raw := range []string{"http://youtube.com/watch?v=abc", "https://youtube.com.evil/watch?v=abc", "https://vimeo.com/12%2Fjavascript"} {
		if _, ok := OEmbedURL(raw); ok {
			t.Fatalf("accepted %q", raw)
		}
	}
}
