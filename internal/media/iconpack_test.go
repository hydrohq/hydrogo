package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateIconPackCreatesStandardImagesAndManifest(t *testing.T) {
	var source bytes.Buffer
	logo := image.NewRGBA(image.Rect(0, 0, 8, 4))
	logo.Set(2, 2, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&source, logo); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := GenerateIconPack(&source, dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"favicon.ico", "apple-touch-icon.png", "icon-192.png", "icon-512.png", "manifest.webmanifest"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || info.Size() < 8 {
			t.Fatalf("%s was not generated: %v", name, err)
		}
	}
	file, err := os.Open(filepath.Join(dir, "icon-192.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	icon, err := png.Decode(file)
	if err != nil || icon.Bounds().Dx() != 192 || icon.Bounds().Dy() != 192 {
		t.Fatalf("decoded icon = %v, %v", icon, err)
	}
}

func TestGenerateIconPackRejectsInvalidImage(t *testing.T) {
	if err := GenerateIconPack(bytes.NewReader([]byte("not an image")), t.TempDir()); err == nil {
		t.Fatal("invalid input accepted")
	}
}
