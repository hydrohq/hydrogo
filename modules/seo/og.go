// Package seo contains provider-neutral public discovery and social metadata.
package seo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	OGWidth  = 1200
	OGHeight = 630
)

type OGInput struct {
	Title    string
	Brand    string
	Category string
}

type OGRenderer struct {
	CacheDir   string
	MaxEntries int
	mu         sync.Mutex
	cache      map[string][]byte
}

func (r *OGRenderer) Render(input OGInput) ([]byte, error) {
	input.Title = boundedText(input.Title, 300)
	input.Brand = boundedText(input.Brand, 100)
	input.Category = boundedText(input.Category, 100)
	if input.Title == "" {
		return nil, fmt.Errorf("OpenGraph title is required")
	}
	key := fmt.Sprintf("%s\x00%s\x00%s", input.Title, input.Brand, input.Category)
	digest := sha256.Sum256([]byte(key))
	name := hex.EncodeToString(digest[:])
	if data, ok := r.cached(name); ok {
		return data, nil
	}
	if r.CacheDir != "" {
		if data, err := os.ReadFile(filepath.Join(r.CacheDir, name+".png")); err == nil && len(data) <= 2<<20 { // #nosec G304 -- name is the hex SHA-256 of the input, so it cannot contain a path
			r.remember(name, data)
			return data, nil
		}
	}
	data, err := renderOG(input)
	if err != nil {
		return nil, err
	}
	r.remember(name, data)
	if r.CacheDir != "" {
		if err := writeCacheFile(r.CacheDir, name+".png", data); err != nil {
			return nil, err
		}
	}
	return append([]byte(nil), data...), nil
}

// OGVersion provides a public, content-derived URL token so cached previews
// refresh when an editorial title changes without exposing the title itself.
func OGVersion(input OGInput) string {
	input.Title = boundedText(input.Title, 300)
	input.Brand = boundedText(input.Brand, 100)
	input.Category = boundedText(input.Category, 100)
	digest := sha256.Sum256([]byte(input.Title + "\x00" + input.Brand + "\x00" + input.Category))
	return hex.EncodeToString(digest[:8])
}

func (r *OGRenderer) cached(key string) ([]byte, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, ok := r.cache[key]
	return append([]byte(nil), data...), ok
}

func (r *OGRenderer) remember(key string, data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cache == nil {
		r.cache = make(map[string][]byte)
	}
	limit := r.MaxEntries
	if limit < 1 || limit > 256 {
		limit = 64
	}
	if len(r.cache) >= limit {
		for old := range r.cache {
			delete(r.cache, old)
			break
		}
	}
	r.cache[key] = append([]byte(nil), data...)
}

func renderOG(input OGInput) ([]byte, error) {
	faceData, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(faceData, &opentype.FaceOptions{Size: 48, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	defer face.Close()
	canvas := image.NewRGBA(image.Rect(0, 0, OGWidth, OGHeight))
	for y := 0; y < OGHeight; y++ {
		t := float64(y) / float64(OGHeight-1)
		line := color.RGBA{R: uint8(16 + 26*t), G: uint8(31 + 37*t), B: uint8(65 + 142*t), A: 255}
		draw.Draw(canvas, image.Rect(0, y, OGWidth, y+1), &image.Uniform{C: line}, image.Point{}, draw.Src)
	}
	white := color.RGBA{R: 250, G: 252, B: 255, A: 255}
	muted := color.RGBA{R: 199, G: 211, B: 236, A: 255}
	drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(white), Face: face}
	if input.Category != "" {
		drawer.Dot = fixed.P(84, 116)
		drawer.DrawString(strings.ToUpper(input.Category))
	}
	lines := wrapText(face, input.Title, 1030)
	fontHeight := face.Metrics().Height.Ceil()
	if len(lines) > 4 {
		lines = lines[:4]
		lines[3] = trimToWidth(face, lines[3], 990) + "…"
	}
	startY := 256 - (len(lines)-1)*fontHeight/2
	for i, line := range lines {
		drawer.Src = image.NewUniform(white)
		drawer.Dot = fixed.P(84, startY+i*fontHeight)
		drawer.DrawString(line)
	}
	if input.Brand != "" {
		drawer.Src = image.NewUniform(muted)
		drawer.Dot = fixed.P(84, 560)
		drawer.DrawString(input.Brand)
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func wrapText(face font.Face, text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, 3)
	line := words[0]
	for _, word := range words[1:] {
		candidate := line + " " + word
		if font.MeasureString(face, candidate).Ceil() > width {
			lines = append(lines, line)
			line = word
		} else {
			line = candidate
		}
	}
	return append(lines, line)
}

func trimToWidth(face font.Face, value string, width int) string {
	for len(value) > 0 && font.MeasureString(face, value).Ceil() > width {
		_, size := utf8DecodeLast(value)
		value = value[:len(value)-size]
	}
	return strings.TrimSpace(value)
}

func utf8DecodeLast(value string) (rune, int) {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i]&0xc0 != 0x80 {
			return []rune(value[i:])[0], len(value) - i
		}
	}
	return 0, len(value)
}

func boundedText(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > max {
		value = value[:max]
		for !utf8Valid(value) && len(value) > 0 {
			value = value[:len(value)-1]
		}
	}
	return value
}

func utf8Valid(value string) bool { return strings.ToValidUTF8(value, "") == value }

func writeCacheFile(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".og-*.tmp")
	if err != nil {
		return err
	}
	path := temporary.Name()
	defer os.Remove(path)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(path, filepath.Join(dir, name))
}
