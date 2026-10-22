package media

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image"
	_ "image/gif"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const maxIconSourceBytes = 20 << 20

var ErrInvalidIconSource = errors.New("invalid icon source image")

// GenerateIconPack creates the standard web app icons from one raster logo.
func GenerateIconPack(source io.Reader, outputDir string) error {
	data, err := io.ReadAll(io.LimitReader(source, maxIconSourceBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxIconSourceBytes {
		return ErrInvalidIconSource
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ErrInvalidIconSource
	}
	bounds := decoded.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 || bounds.Dx() > 8192 || bounds.Dy() > 8192 || int64(bounds.Dx())*int64(bounds.Dy()) > 24_000_000 {
		return ErrInvalidIconSource
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil { // #nosec G301 -- public static asset directory
		return err
	}
	for _, size := range []int{180, 192, 512, 32} {
		canvas := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.ApproxBiLinear.Scale(canvas, canvas.Bounds(), decoded, bounds, draw.Over, nil)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, canvas); err != nil {
			return err
		}
		name := map[int]string{180: "apple-touch-icon.png", 192: "icon-192.png", 512: "icon-512.png"}[size]
		if size == 32 {
			if err := writeICO(filepath.Join(outputDir, "favicon.ico"), encoded.Bytes()); err != nil {
				return err
			}
			continue
		}
		if err := os.WriteFile(filepath.Join(outputDir, name), encoded.Bytes(), 0o644); err != nil { // #nosec G306 -- public static asset served to browsers
			return err
		}
	}
	manifest := struct {
		Name      string              `json:"name"`
		ShortName string              `json:"short_name"`
		StartURL  string              `json:"start_url"`
		Display   string              `json:"display"`
		Icons     []map[string]string `json:"icons"`
	}{Name: "HydroGo", ShortName: "HydroGo", StartURL: "/", Display: "standalone", Icons: []map[string]string{{"src": "/icon-192.png", "sizes": "192x192", "type": "image/png"}, {"src": "/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any maskable"}}}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "manifest.webmanifest"), manifestData, 0o644) // #nosec G306 -- public static asset served to browsers
}

func writeICO(path string, pngData []byte) error {
	if len(pngData) == 0 || uint64(len(pngData)) > uint64(^uint32(0))-22 {
		return ErrInvalidIconSource
	}
	var ico bytes.Buffer
	_, _ = ico.Write([]byte{0, 0, 1, 0, 1, 0, 32, 32, 0, 0, 1, 0, 32, 0})
	_ = binary.Write(&ico, binary.LittleEndian, uint32(len(pngData))) // #nosec G115 -- a PNG of a 32x32 icon is far below 4 GiB
	_ = binary.Write(&ico, binary.LittleEndian, uint32(22))
	_, _ = ico.Write(pngData)
	return os.WriteFile(path, ico.Bytes(), 0o644) // #nosec G306 -- public static asset served to browsers
}
