package transport

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream != nil {
		h.streamSitemap(w, r, "", false)
		return
	}
	entries, ok := h.sitemapEntries(w, r)
	if !ok {
		return
	}
	parts, err := site.SplitSitemap(entries, site.MaxSitemapURLs, site.MaxSitemapBytes)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	if part := strings.TrimSpace(r.URL.Query().Get("part")); part != "" {
		index, parseErr := strconv.Atoi(part)
		if parseErr != nil || index < 1 || index > len(parts) {
			http.NotFound(w, r)
			return
		}
		writeSitemap(w, parts[index-1])
		return
	}
	if len(parts) > 1 {
		data, err := site.RenderSitemapIndex(h.SitemapOrigin, len(parts))
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		writeSitemapData(w, data)
		return
	}
	writeSitemap(w, entries)
}

func (h Handler) sitemapIndex(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream != nil {
		h.streamSitemap(w, r, "", true)
		return
	}
	entries, ok := h.sitemapEntries(w, r)
	if !ok {
		return
	}
	parts, err := site.SplitSitemap(entries, site.MaxSitemapURLs, site.MaxSitemapBytes)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	data, err := site.RenderSitemapIndex(h.SitemapOrigin, len(parts))
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	writeSitemapData(w, data)
}

func (h Handler) sitemapPart(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream != nil {
		h.streamSitemap(w, r, "pages", false)
		return
	}
	part, err := strconv.Atoi(r.URL.Query().Get("part"))
	if err != nil || part < 1 {
		http.NotFound(w, r)
		return
	}
	entries, ok := h.sitemapEntries(w, r)
	if !ok {
		return
	}
	parts, err := site.SplitSitemap(entries, site.MaxSitemapURLs, site.MaxSitemapBytes)
	if err != nil || part > len(parts) {
		http.NotFound(w, r)
		return
	}
	writeSitemap(w, parts[part-1])
}

func (h Handler) sitemapArticlePart(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream == nil {
		http.NotFound(w, r)
		return
	}
	h.streamSitemap(w, r, "articles", false)
}

func (h Handler) streamSitemap(w http.ResponseWriter, r *http.Request, category string, forceIndex bool) {
	if h.SitemapOrigin == "" {
		http.Error(w, "sitemap unavailable", http.StatusServiceUnavailable)
		return
	}
	if category == "" && strings.TrimSpace(r.URL.Query().Get("part")) != "" {
		category = "pages"
	}
	categories := h.SitemapCategories
	if len(categories) == 0 {
		categories = []string{"pages"}
	}
	if category != "" {
		part := 1
		if raw := strings.TrimSpace(r.URL.Query().Get("part")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				http.NotFound(w, r)
				return
			}
			part = parsed
		}
		var output bytes.Buffer
		count, err := site.StreamSitemapPart(r.Context(), h.SitemapStream, category, part, site.MaxSitemapURLs, site.MaxSitemapBytes, &output)
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		if part > count || count == 0 {
			http.NotFound(w, r)
			return
		}
		writeSitemapData(w, output.Bytes())
		return
	}
	counts := make(map[string]int, len(categories))
	totalParts := 0
	for _, item := range categories {
		count, err := site.StreamSitemapPart(r.Context(), h.SitemapStream, item, 0, site.MaxSitemapURLs, site.MaxSitemapBytes, nil)
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			counts[item] = count
			totalParts += count
		}
	}
	if totalParts == 0 {
		writeSitemap(w, nil)
		return
	}
	if forceIndex || totalParts > 1 {
		data, err := site.RenderCategorizedSitemapIndex(h.SitemapOrigin, counts)
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		writeSitemapData(w, data)
		return
	}
	for item, count := range counts {
		if count == 1 {
			var output bytes.Buffer
			_, err := site.StreamSitemapPart(r.Context(), h.SitemapStream, item, 1, site.MaxSitemapURLs, site.MaxSitemapBytes, &output)
			if err != nil {
				http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
				return
			}
			writeSitemapData(w, output.Bytes())
			return
		}
	}
}

func (h Handler) sitemapEntries(w http.ResponseWriter, r *http.Request) ([]site.SitemapEntry, bool) {
	if h.SitemapProvider == nil || h.SitemapOrigin == "" {
		http.Error(w, "sitemap unavailable", http.StatusServiceUnavailable)
		return nil, false
	}
	entries, err := h.SitemapProvider(r.Context())
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return nil, false
	}
	return entries, true
}

func writeSitemap(w http.ResponseWriter, entries []site.SitemapEntry) {
	data, err := site.RenderSitemap(entries)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	writeSitemapData(w, data)
}

func writeSitemapData(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=300")
	_, _ = w.Write(data) // #nosec G705 -- sitemap XML the server generated and escaped, sent as application/xml
}
