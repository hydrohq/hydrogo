package site

import (
	"encoding/xml"
	"errors"
	"net/url"
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
}

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
