package media

import (
	"errors"
	"fmt"
	"golang.org/x/text/unicode/norm"
	"image"
	"image/color"
	"math"
	"net/url"
	"sort"
	"strings"
	"unicode"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	imagedraw "image/draw"
)

// FocalPoint is a normalized image coordinate. Values are clamped to [0,1].
type FocalPoint struct{ X, Y float64 }

func (p FocalPoint) ObjectPosition() string {
	return fmt.Sprintf("%.2f%% %.2f%%", clamp01(p.X)*100, clamp01(p.Y)*100)
}

// CropImage fits a target aspect ratio around the focal point and scales to size.
func CropImage(src image.Image, width, height int, focal FocalPoint) (image.Image, error) {
	if src == nil || width < 1 || height < 1 {
		return nil, errors.New("invalid crop dimensions")
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw < 1 || sh < 1 {
		return nil, errors.New("empty source image")
	}
	ratio := float64(width) / float64(height)
	cw, ch := sw, int(math.Round(float64(sw)/ratio))
	if ch > sh {
		ch, cw = sh, int(math.Round(float64(sh)*ratio))
	}
	x := int(math.Round(clamp01(focal.X)*float64(sw-1) - float64(cw)/2))
	y := int(math.Round(clamp01(focal.Y)*float64(sh-1) - float64(ch)/2))
	x = max(0, min(sw-cw, x))
	y = max(0, min(sh-ch, y))
	area := image.NewRGBA(image.Rect(0, 0, cw, ch))
	imagedraw.Draw(area, area.Bounds(), src, b.Min.Add(image.Pt(x, y)), imagedraw.Src)
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(out, out.Bounds(), area, area.Bounds(), xdraw.Over, nil)
	return out, nil
}

// Watermark overlays a logo and/or text on a fresh image, preserving the source.
type WatermarkOptions struct {
	Position string
	Opacity  float64
	Margin   int
	Logo     image.Image
	Text     string
}

func ApplyWatermark(src image.Image, opts WatermarkOptions) (image.Image, error) {
	if src == nil {
		return nil, errors.New("source image is required")
	}
	if opts.Position == "" {
		opts.Position = "bottom-right"
	}
	if opts.Position != "bottom-right" && opts.Position != "bottom-left" && opts.Position != "top-right" && opts.Position != "center" {
		return nil, errors.New("invalid watermark position")
	}
	if opts.Opacity == 0 {
		opts.Opacity = 1
	}
	if opts.Opacity < .1 || opts.Opacity > 1 {
		return nil, errors.New("watermark opacity must be between 0.1 and 1")
	}
	if opts.Margin < 0 || opts.Margin > 1000 {
		return nil, errors.New("invalid watermark margin")
	}
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	imagedraw.Draw(out, out.Bounds(), src, b.Min, imagedraw.Src)
	if opts.Logo != nil {
		lb := opts.Logo.Bounds()
		w, h := lb.Dx(), lb.Dy()
		if w > b.Dx()/2 {
			h = h * b.Dx() / 2 / w
			w = b.Dx() / 2
		}
		if h > b.Dy()/2 {
			w = w * b.Dy() / 2 / h
			h = b.Dy() / 2
		}
		dx, dy := watermarkPoint(b.Dx(), b.Dy(), w, h, opts.Margin, opts.Position)
		logo := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(logo, logo.Bounds(), opts.Logo, lb, xdraw.Over, nil)
		for i := range logo.Pix {
			if i%4 == 3 {
				logo.Pix[i] = uint8(float64(logo.Pix[i]) * opts.Opacity)
			}
		}
		imagedraw.Draw(out, image.Rect(dx, dy, dx+w, dy+h), logo, image.Point{}, imagedraw.Over)
	}
	if opts.Text != "" {
		face := basicfont.Face7x13
		width := font.MeasureString(face, opts.Text).Ceil()
		x, y := watermarkPoint(b.Dx(), b.Dy(), width, 13, opts.Margin, opts.Position)
		d := font.Drawer{Dst: out, Src: image.NewUniform(color.NRGBA{255, 255, 255, uint8(255 * opts.Opacity)}), Face: face, Dot: fixed.P(x, y+12)}
		d.DrawString(opts.Text)
	}
	return out, nil
}
func watermarkPoint(w, h, ow, oh, m int, pos string) (int, int) {
	switch pos {
	case "bottom-right":
		return max(0, w-ow-m), max(0, h-oh-m)
	case "bottom-left":
		return m, max(0, h-oh-m)
	case "top-right":
		return max(0, w-ow-m), m
	default:
		return max(0, (w-ow)/2), max(0, (h-oh)/2)
	}
}

// DominantColors samples a bounded grid and returns the most common colors.
func DominantColors(src image.Image, count int) []string {
	if src == nil || count < 1 {
		return nil
	}
	if count > 5 {
		count = 5
	}
	b := src.Bounds()
	hist := map[uint32]int{}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			c := color.NRGBAModel.Convert(src.At(b.Min.X+(x*b.Dx())/32, b.Min.Y+(y*b.Dy())/32)).(color.NRGBA)
			if c.A < 128 {
				continue
			}
			r, g, bl := uint32(c.R>>3), uint32(c.G>>3), uint32(c.B>>3)
			hist[r<<10|g<<5|bl]++
		}
	}
	type pair struct {
		k uint32
		n int
	}
	values := make([]pair, 0, len(hist))
	for k, n := range hist {
		values = append(values, pair{k, n})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].n == values[j].n {
			return values[i].k < values[j].k
		}
		return values[i].n > values[j].n
	})
	out := make([]string, 0, min(count, len(values)))
	for _, v := range values {
		r, g, b := uint8(((v.k>>10)&31)*255/31), uint8(((v.k>>5)&31)*255/31), uint8((v.k&31)*255/31)
		s := fmt.Sprintf("#%02x%02x%02x", r, g, b)
		duplicate := false
		for _, old := range out {
			if old == s {
				duplicate = true
			}
		}
		if !duplicate {
			out = append(out, s)
		}
		if len(out) == count {
			break
		}
	}
	return out
}

const blurAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz#$%*+,-.:;=?@[]^_{|}~"

// BlurHash computes a compact, standard BlurHash string from an image.
func BlurHash(src image.Image, componentsX, componentsY int) (string, error) {
	if src == nil || componentsX < 1 || componentsX > 9 || componentsY < 1 || componentsY > 9 {
		return "", errors.New("invalid blurhash input")
	}
	b := src.Bounds()
	if b.Empty() {
		return "", errors.New("empty image")
	}
	factors := make([][3]float64, componentsX*componentsY)
	maxv := 0.0
	const samples = 32
	for j := 0; j < componentsY; j++ {
		for i := 0; i < componentsX; i++ {
			var r, g, bl float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					x := b.Min.X + sx*b.Dx()/samples
					y := b.Min.Y + sy*b.Dy()/samples
					c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
					f := basis(i, sx, sy, samples, samples)
					r += srgbLinear(c.R) * f
					g += srgbLinear(c.G) * f
					bl += srgbLinear(c.B) * f
				}
			}
			scale := 1 / float64(samples*samples)
			factors[j*componentsX+i] = [3]float64{r * scale, g * scale, bl * scale}
			if i != 0 || j != 0 {
				maxv = math.Max(maxv, math.Max(math.Abs(r*scale), math.Max(math.Abs(g*scale), math.Abs(bl*scale))))
			}
		}
	}
	maximum := int(math.Max(0, math.Min(82, math.Floor(maxv*166-0.5))))
	quant := math.Floor(float64(maximum+1)/166*1000) / 1000
	var out strings.Builder
	out.WriteString(encode83((componentsX-1)+(componentsY-1)*9, 1))
	out.WriteString(encode83(maximum, 1))
	out.WriteString(encode83(encodeDC(factors[0]), 4))
	for _, f := range factors[1:] {
		out.WriteString(encode83(encodeAC(f, quant), 2))
	}
	return out.String(), nil
}
func basis(i, x, y, w, h int) float64 {
	scale := math.Cos(math.Pi*float64(i)*float64(x)/float64(w)) * math.Cos(math.Pi*float64(y)/float64(h))
	return scale
}
func srgbLinear(v uint8) float64 {
	x := float64(v) / 255
	if x <= .04045 {
		return x / 12.92
	}
	return math.Pow((x+.055)/1.055, 2.4)
}
func encodeDC(f [3]float64) int {
	r, g, b := linearSRGB(f[0]), linearSRGB(f[1]), linearSRGB(f[2])
	return (r << 16) + (g << 8) + b
}
func linearSRGB(v float64) int {
	x := 0
	if v <= .0031308 {
		x = int(v*12.92*255 + .5)
	} else {
		x = int((1.055*math.Pow(v, 1/2.4)-.055)*255 + .5)
	}
	return max(0, min(255, x))
}
func encodeAC(f [3]float64, m float64) int {
	q := func(v float64) int {
		x := math.Copysign(math.Floor(math.Sqrt(math.Abs(v)/m)+.5), v)
		return max(0, min(18, int(x)+9))
	}
	return q(f[0])*19*19 + q(f[1])*19 + q(f[2])
}
func encode83(v, n int) string {
	var b strings.Builder
	for i := n - 1; i >= 0; i-- {
		b.WriteByte(blurAlphabet[(v/int(math.Pow(83, float64(i))))%83])
	}
	return b.String()
}

// BlurHashColor returns the average color encoded by the BlurHash DC term.
func BlurHashColor(hash string) string {
	if len(hash) < 6 {
		return "#e5e7eb"
	}
	decode := func(s string) int {
		n := 0
		for _, r := range s {
			i := strings.IndexRune(blurAlphabet, r)
			if i < 0 {
				return -1
			}
			n = n*83 + i
		}
		return n
	}
	n := decode(hash[2:6])
	if n < 0 {
		return "#e5e7eb"
	}
	return fmt.Sprintf("#%02x%02x%02x", (n>>16)&255, (n>>8)&255, n&255)
}
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// NormalizeFilename returns an accent-free, URL-safe filename slug.
func NormalizeFilename(value string) string {
	value = strings.TrimSpace(value)
	ext := strings.ToLower(pathExt(value))
	stem := strings.TrimSuffix(value, pathExt(value))
	var out strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(stem)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == 'ı' {
			r = 'i'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			dash = false
		} else if !dash && out.Len() > 0 {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-") + ext
}
func pathExt(s string) string {
	i := strings.LastIndexByte(s, '.')
	if i < 0 || i == 0 {
		return ""
	}
	return s[i:]
}

// OEmbedURL returns a privacy-enhanced, allowlisted provider URL.
func OEmbedURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	id := ""
	provider := ""
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		}
		if strings.HasPrefix(u.Path, "/shorts/") {
			id = strings.TrimPrefix(u.Path, "/shorts/")
		}
		provider = "https://www.youtube-nocookie.com/embed/"
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
		provider = "https://www.youtube-nocookie.com/embed/"
	case "vimeo.com", "www.vimeo.com":
		id = strings.Trim(u.Path, "/")
		provider = "https://player.vimeo.com/video/"
	}
	if id == "" || len(id) > 64 {
		return "", false
	}
	for _, r := range id {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			return "", false
		}
	}
	return provider + id, true
}
