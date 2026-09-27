package seo

import (
	"bytes"
	"image/png"
	"path/filepath"
	"testing"
)

func TestOGRendererProduces1200By630PNGAndCaches(t *testing.T) {
	renderer := &OGRenderer{CacheDir: t.TempDir()}
	input := OGInput{Title: "Hizmetlerimiz ve Çözümlerimiz", Brand: "Örnek Şirket", Category: "Hizmetler"}
	first, err := renderer.Render(input)
	if err != nil {
		t.Fatal(err)
	}
	image, err := png.Decode(bytes.NewReader(first))
	if err != nil || image.Bounds().Dx() != OGWidth || image.Bounds().Dy() != OGHeight {
		t.Fatalf("OG image bounds=%v, err=%v", image.Bounds(), err)
	}
	second, err := renderer.Render(input)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("cached OG render differs, err=%v", err)
	}
	files, err := filepath.Glob(filepath.Join(renderer.CacheDir, "*.png"))
	if err != nil || len(files) != 1 {
		t.Fatalf("disk cache files=%v, err=%v", files, err)
	}
}

func TestOGRendererKeysCacheByMetadataAndRequiresTitle(t *testing.T) {
	renderer := &OGRenderer{}
	first, err := renderer.Render(OGInput{Title: "First title", Brand: "Brand"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := renderer.Render(OGInput{Title: "Updated title", Brand: "Brand"})
	if err != nil || bytes.Equal(first, second) {
		t.Fatalf("title change did not invalidate the image: err=%v", err)
	}
	if _, err := renderer.Render(OGInput{}); err == nil {
		t.Fatal("empty title accepted")
	}
	if OGVersion(OGInput{Title: "First title", Brand: "Brand"}) == OGVersion(OGInput{Title: "Updated title", Brand: "Brand"}) {
		t.Fatal("editorial title changes must produce a new public cache URL")
	}
}
