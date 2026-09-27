package site

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

var ErrInvalidSEOOrigin = errors.New("invalid SEO origin")

type SEOInput struct {
	Origin        string
	Locale        string
	Path          string
	Title         string
	Description   string
	OGTitle       string
	OGDescription string
	OGImage       string
}

type SEOView struct {
	CanonicalURL  string
	Title         string
	Description   string
	OGTitle       string
	OGDescription string
	OGImage       string
}

func BuildSEO(input SEOInput) (SEOView, error) {
	origin, err := validOrigin(input.Origin)
	if err != nil {
		return SEOView{}, err
	}
	path := normalizePath(input.Path)
	if input.Locale != "" {
		path = "/" + canonical.Locale(input.Locale) + path
	}
	canonical := strings.TrimRight(origin, "/") + encodePath(path)
	title := strings.TrimSpace(input.Title)
	ogTitle := strings.TrimSpace(input.OGTitle)
	if ogTitle == "" {
		ogTitle = title
	}
	ogDescription := strings.TrimSpace(input.OGDescription)
	if ogDescription == "" {
		ogDescription = strings.TrimSpace(input.Description)
	}
	return SEOView{CanonicalURL: canonical, Title: title, Description: strings.TrimSpace(input.Description), OGTitle: ogTitle, OGDescription: ogDescription, OGImage: strings.TrimSpace(input.OGImage)}, nil
}

type SitemapEntry struct {
	URL          string
	LastModified time.Time
	Category     string
}

// SitemapStream visits published URLs in stable order without retaining the
// full catalog in memory. The category is one of the registered public sets.
type SitemapStream func(context.Context, string, func(SitemapEntry) error) error

const (
	MaxSitemapURLs  = 50000
	MaxSitemapBytes = 50 << 20
)

type sitemapURLSet struct {
	XMLName xml.Name      `xml:"urlset"`
	XMLNS   string        `xml:"xmlns,attr"`
	URLs    []sitemapNode `xml:"url"`
}

type sitemapNode struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func RenderSitemap(entries []SitemapEntry) ([]byte, error) {
	result := sitemapURLSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: make([]sitemapNode, 0, len(entries))}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		parsed, err := url.ParseRequestURI(entry.URL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, ErrInvalidSEOOrigin
		}
		if _, ok := seen[entry.URL]; ok {
			continue
		}
		seen[entry.URL] = struct{}{}
		node := sitemapNode{Loc: entry.URL}
		if !entry.LastModified.IsZero() {
			node.LastMod = entry.LastModified.UTC().Format("2006-01-02")
		}
		result.URLs = append(result.URLs, node)
	}
	return xml.Marshal(result)
}

type sitemapIndex struct {
	XMLName xml.Name          `xml:"sitemapindex"`
	XMLNS   string            `xml:"xmlns,attr"`
	Files   []sitemapIndexURL `xml:"sitemap"`
}

type sitemapIndexURL struct {
	Location string `xml:"loc"`
}

// SplitSitemap keeps every protocol file below both standard URL and byte caps.
// Callers can lower the caps in tests; zero values use the sitemap defaults.
func SplitSitemap(entries []SitemapEntry, maxURLs, maxBytes int) ([][]SitemapEntry, error) {
	if maxURLs < 1 || maxURLs > MaxSitemapURLs {
		maxURLs = MaxSitemapURLs
	}
	if maxBytes < 256 || maxBytes > MaxSitemapBytes {
		maxBytes = MaxSitemapBytes
	}
	parts := make([][]SitemapEntry, 0, len(entries)/maxURLs+1)
	current := make([]SitemapEntry, 0, min(maxURLs, len(entries)))
	bytesUsed := 128
	for _, entry := range entries {
		parsed, err := url.ParseRequestURI(entry.URL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, ErrInvalidSEOOrigin
		}
		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(entry.URL)); err != nil {
			return nil, err
		}
		estimate := escaped.Len() + 64
		if estimate+128 > maxBytes {
			return nil, ErrInvalidSEOOrigin
		}
		if len(current) > 0 && (len(current) == maxURLs || bytesUsed+estimate > maxBytes) {
			parts = append(parts, current)
			current = make([]SitemapEntry, 0, min(maxURLs, len(entries)))
			bytesUsed = 128
		}
		current = append(current, entry)
		bytesUsed += estimate
	}
	if len(current) > 0 || len(parts) == 0 {
		parts = append(parts, current)
	}
	return parts, nil
}

func RenderSitemapIndex(origin string, parts int) ([]byte, error) {
	return RenderCategorizedSitemapIndex(origin, map[string]int{"pages": parts})
}

func RenderCategorizedSitemapIndex(origin string, categories map[string]int) ([]byte, error) {
	base, err := validOrigin(origin)
	if err != nil || len(categories) == 0 {
		return nil, ErrInvalidSEOOrigin
	}
	keys := make([]string, 0, len(categories))
	total := 0
	for category, parts := range categories {
		if !validSitemapCategory(category) || parts < 0 || parts > 10000 {
			return nil, ErrInvalidSEOOrigin
		}
		if parts > 0 {
			keys = append(keys, category)
			total += parts
		}
	}
	if total == 0 || total > MaxSitemapURLs {
		return nil, ErrInvalidSEOOrigin
	}
	sort.Strings(keys)
	result := sitemapIndex{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", Files: make([]sitemapIndexURL, 0, total)}
	for _, category := range keys {
		for part := 1; part <= categories[category]; part++ {
			result.Files = append(result.Files, sitemapIndexURL{Location: base + "/sitemap-" + category + ".xml?part=" + strconv.Itoa(part)})
		}
	}
	return xml.Marshal(result)
}

func validSitemapCategory(category string) bool {
	if category == "" || len(category) > 32 {
		return false
	}
	for _, char := range category {
		if char < 'a' || char > 'z' {
			return false
		}
	}
	return true
}

// StreamSitemapPart emits one protocol-bounded URL set while counting all
// parts. targetPart=0 counts only; callers can build an index without buffering
// or materializing the catalog. The writer receives only the requested part.
func StreamSitemapPart(ctx context.Context, stream SitemapStream, category string, targetPart, maxURLs, maxBytes int, output io.Writer) (int, error) {
	if stream == nil || !validSitemapCategory(category) || targetPart < 0 {
		return 0, ErrInvalidSEOOrigin
	}
	if maxURLs < 1 || maxURLs > MaxSitemapURLs {
		maxURLs = MaxSitemapURLs
	}
	if maxBytes < 256 || maxBytes > MaxSitemapBytes {
		maxBytes = MaxSitemapBytes
	}
	const open = `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`
	const close = `</urlset>`
	part, count, used := 0, 0, len(open)+len(close)
	var buffer bytes.Buffer
	beginPart := func() {
		part++
		count = 0
		used = len(open) + len(close)
		buffer.Reset()
		if targetPart == part {
			buffer.WriteString(open)
		}
	}
	finishPart := func() error {
		if count == 0 {
			return nil
		}
		if targetPart == part && output != nil {
			buffer.WriteString(close)
			_, err := output.Write(buffer.Bytes())
			return err
		}
		return nil
	}
	err := stream(ctx, category, func(entry SitemapEntry) error {
		if !validSitemapEntry(entry.URL) {
			return ErrInvalidSEOOrigin
		}
		node, err := marshalSitemapNode(entry)
		if err != nil {
			return err
		}
		if len(open)+len(close)+len(node) > maxBytes {
			return ErrInvalidSEOOrigin
		}
		if count > 0 && (count >= maxURLs || used+len(node) > maxBytes) {
			if err := finishPart(); err != nil {
				return err
			}
			count = 0
			used = len(open) + len(close)
			buffer.Reset()
		}
		if count == 0 {
			beginPart()
		}
		if targetPart == part {
			buffer.Write(node)
		}
		count++
		used += len(node)
		return nil
	})
	if err != nil {
		return 0, err
	}
	if err := finishPart(); err != nil {
		return 0, err
	}
	return part, nil
}

func validSitemapEntry(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func marshalSitemapNode(entry SitemapEntry) ([]byte, error) {
	var result bytes.Buffer
	result.WriteString("<url><loc>")
	if err := xml.EscapeText(&result, []byte(entry.URL)); err != nil {
		return nil, err
	}
	result.WriteString("</loc>")
	if !entry.LastModified.IsZero() {
		result.WriteString("<lastmod>")
		if err := xml.EscapeText(&result, []byte(entry.LastModified.UTC().Format("2006-01-02"))); err != nil {
			return nil, err
		}
		result.WriteString("</lastmod>")
	}
	result.WriteString("</url>")
	return result.Bytes(), nil
}

func validOrigin(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", ErrInvalidSEOOrigin
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func encodePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
