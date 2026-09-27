package designsystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
)

const (
	maxFontCSSBytes  = 256 << 10
	maxFontFileBytes = 2 << 20
	maxFontFaces     = 32
)

var (
	fontFamilyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _-]{0,79}$`)
	fontFacePattern   = regexp.MustCompile(`(?s)@font-face\s*\{([^}]*)\}`)
	fontURLPattern    = regexp.MustCompile(`url\((?:"|')?(https://fonts\.gstatic\.com/[^)"']+)(?:"|')?\)`)
	fontFamilyCSS     = regexp.MustCompile(`font-family\s*:\s*["']?([^;"']+)`)
	fontWeightCSS     = regexp.MustCompile(`font-weight\s*:\s*([0-9 ]+(?:\.\.[0-9]+)?)`)
	fontStyleCSS      = regexp.MustCompile(`font-style\s*:\s*(normal|italic|oblique)`)
	fontRangeCSS      = regexp.MustCompile(`unicode-range\s*:\s*([Uu]\+[0-9A-Fa-f?, -]+)`)
)

// LocalizeGoogleFont downloads WOFF2 faces from the fixed Google Fonts CSS API
// and stores only allowlisted fonts.gstatic.com resources under destination.
func LocalizeGoogleFont(ctx context.Context, family, destination string, client *http.Client) (string, error) {
	family = strings.TrimSpace(family)
	if !fontFamilyPattern.MatchString(family) || strings.TrimSpace(destination) == "" {
		return "", errors.New("font family and destination are required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 2 || (req.URL.Host != "fonts.googleapis.com" && req.URL.Host != "fonts.gstatic.com") {
				return http.ErrUseLastResponse
			}
			return nil
		}}
	}
	endpoint := "https://fonts.googleapis.com/css2?family=" + url.QueryEscape(family+":wght@400..700") + "&display=swap"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 HydroGo Font Localizer")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || (response.Request != nil && response.Request.URL.Host != "fonts.googleapis.com") {
		return "", errors.New("Google Fonts stylesheet unavailable")
	}
	css, err := io.ReadAll(io.LimitReader(response.Body, maxFontCSSBytes+1))
	if err != nil || len(css) > maxFontCSSBytes {
		return "", errors.New("Google Fonts stylesheet exceeds size limit")
	}
	faces := fontFacePattern.FindAllSubmatch(css, maxFontFaces+1)
	if len(faces) == 0 || len(faces) > maxFontFaces {
		return "", errors.New("Google Fonts stylesheet has no supported font faces")
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return "", err
	}
	if err := pathguard.RejectSymlinkComponents(destination); err != nil {
		return "", err
	}
	var output strings.Builder
	for _, match := range faces {
		declaration := string(match[1])
		fontURL := fontURLPattern.FindStringSubmatch(declaration)
		familyMatch := fontFamilyCSS.FindStringSubmatch(declaration)
		weightMatch := fontWeightCSS.FindStringSubmatch(declaration)
		styleMatch := fontStyleCSS.FindStringSubmatch(declaration)
		if len(fontURL) != 2 || len(familyMatch) != 2 || len(weightMatch) != 2 {
			continue
		}
		assetURL, err := url.Parse(fontURL[1])
		if err != nil || assetURL.Scheme != "https" || assetURL.Host != "fonts.gstatic.com" || assetURL.User != nil {
			continue
		}
		assetRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL.String(), nil)
		if err != nil {
			return "", err
		}
		assetResponse, err := client.Do(assetRequest)
		if err != nil {
			return "", err
		}
		if assetResponse.StatusCode != http.StatusOK || (assetResponse.Request != nil && assetResponse.Request.URL.Host != "fonts.gstatic.com") {
			assetResponse.Body.Close()
			return "", errors.New("font file unavailable")
		}
		data, readErr := io.ReadAll(io.LimitReader(assetResponse.Body, maxFontFileBytes+1))
		assetResponse.Body.Close()
		if readErr != nil || len(data) > maxFontFileBytes {
			return "", errors.New("font file exceeds size limit")
		}
		digest := sha256.Sum256(data)
		filename := "font-" + hex.EncodeToString(digest[:12]) + ".woff2"
		if err := writeFontFile(destination, filename, data); err != nil {
			return "", err
		}
		weight := strings.Join(strings.Fields(weightMatch[1]), " ")
		style := "normal"
		if len(styleMatch) == 2 {
			style = styleMatch[1]
		}
		faceFamily := strings.TrimSpace(strings.Trim(familyMatch[1], "\"'"))
		if !fontFamilyPattern.MatchString(faceFamily) {
			faceFamily = family
		}
		fmt.Fprintf(&output, "@font-face{font-family:'%s';font-style:%s;font-weight:%s;font-display:swap;src:url('./%s') format('woff2');", faceFamily, style, weight, filename)
		if unicodeRange := fontRangeCSS.FindStringSubmatch(declaration); len(unicodeRange) == 2 {
			fmt.Fprintf(&output, "unicode-range:%s;", strings.TrimSpace(unicodeRange[1]))
		}
		output.WriteString("}\n")
	}
	if output.Len() == 0 {
		return "", errors.New("Google Fonts stylesheet has no supported WOFF2 faces")
	}
	fmt.Fprintf(&output, "body{font-family:'%s',sans-serif;}\n", family)
	slug := strings.ToLower(strings.ReplaceAll(strings.Join(strings.Fields(family), "-"), " ", "-"))
	filename := "font-" + slug + ".css"
	if err := writeFontFile(destination, filename, []byte(output.String())); err != nil {
		return "", err
	}
	return filepath.Join(destination, filename), nil
}

// FontAssets serves only the deterministic font files generated by
// LocalizeGoogleFont; arbitrary files in the selected directory stay private.
func FontAssets(directory string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.PathValue("path"), "/")
		if path.Base(name) != name || !(strings.HasPrefix(name, "font-") && (strings.HasSuffix(name, ".woff2") || strings.HasSuffix(name, ".css"))) {
			http.NotFound(w, r)
			return
		}
		if directory == "" || pathguard.RejectSymlinkComponents(directory) != nil {
			http.NotFound(w, r)
			return
		}
		filePath := filepath.Join(directory, name)
		info, err := os.Lstat(filePath)
		maxBytes := int64(maxFontFileBytes)
		if strings.HasSuffix(name, ".css") {
			maxBytes = maxFontCSSBytes
		}
		if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxBytes {
			http.NotFound(w, r)
			return
		}
		file, err := os.Open(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		contentType := mime.TypeByExtension(filepath.Ext(name))
		if strings.HasSuffix(name, ".woff2") {
			contentType = "font/woff2"
		}
		w.Header().Set("Content-Type", contentType)
		if strings.HasSuffix(name, ".woff2") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		http.ServeContent(w, r, name, info.ModTime(), file)
	})
}

func writeFontFile(directory, name string, data []byte) error {
	if filepath.Base(name) != name {
		return errors.New("invalid font output name")
	}
	temporary, err := os.CreateTemp(directory, ".font-*.tmp")
	if err != nil {
		return err
	}
	path := temporary.Name()
	defer os.Remove(path)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(path, filepath.Join(directory, name))
}
