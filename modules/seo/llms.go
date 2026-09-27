package seo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	xhtml "golang.org/x/net/html"
)

const maxLLMSBytes = 2 << 20

type DiscoveryPage struct {
	Title       string
	Path        string
	Description string
	Text        string
}

type Discovery struct {
	Name        string
	Description string
	Origin      string
	Pages       []DiscoveryPage
	ExcludePath []string
}

type DiscoveryProvider func(context.Context) (Discovery, error)

type DiscoveryCache struct {
	mu         sync.Mutex
	generation uint64
	validUntil time.Time
	index      []byte
	full       []byte
	TTL        time.Duration
}

func (c *DiscoveryCache) Invalidate() {
	c.mu.Lock()
	c.generation++
	c.validUntil = time.Time{}
	c.index, c.full = nil, nil
	c.mu.Unlock()
}

func (c *DiscoveryCache) Get(ctx context.Context, provider DiscoveryProvider, full bool) ([]byte, error) {
	if provider == nil {
		return nil, fmt.Errorf("discovery provider is required")
	}
	now := time.Now()
	c.mu.Lock()
	generation := c.generation
	if now.Before(c.validUntil) {
		cached := c.index
		if full {
			cached = c.full
		}
		if cached != nil {
			result := append([]byte(nil), cached...)
			c.mu.Unlock()
			return result, nil
		}
	}
	c.mu.Unlock()
	data, err := provider(ctx)
	if err != nil {
		return nil, err
	}
	index, complete, err := RenderLLMS(data)
	if err != nil {
		return nil, err
	}
	ttl := c.TTL
	if ttl <= 0 || ttl > 10*time.Minute {
		ttl = time.Minute
	}
	c.mu.Lock()
	if c.generation == generation {
		c.index, c.full, c.validUntil = append([]byte(nil), index...), append([]byte(nil), complete...), now.Add(ttl)
	}
	result := index
	if full {
		result = complete
	}
	result = append([]byte(nil), result...)
	c.mu.Unlock()
	return result, nil
}

func RenderLLMS(discovery Discovery) (index, full []byte, err error) {
	name := cleanMarkdown(discovery.Name, 180)
	if name == "" {
		name = "Website"
	}
	summary := cleanMarkdown(discovery.Description, 1200)
	var short, complete strings.Builder
	fmt.Fprintf(&short, "# %s\n\n", name)
	fmt.Fprintf(&complete, "# %s\n\n", name)
	if summary != "" {
		fmt.Fprintf(&short, "> %s\n\n", summary)
		fmt.Fprintf(&complete, "> %s\n\n", summary)
	}
	short.WriteString("## Pages\n\n")
	complete.WriteString("## Pages\n\n")
	count := 0
	for _, page := range discovery.Pages {
		path := cleanPath(page.Path)
		if path == "" || excluded(path, discovery.ExcludePath) {
			continue
		}
		title := cleanMarkdown(page.Title, 180)
		if title == "" {
			continue
		}
		if count >= 10000 {
			break
		}
		count++
		link := path
		if origin := strings.TrimRight(discovery.Origin, "/"); origin != "" {
			link = origin + path
		}
		fmt.Fprintf(&short, "- [%s](%s)", title, link)
		fmt.Fprintf(&complete, "- [%s](%s)", title, link)
		if description := cleanMarkdown(page.Description, 500); description != "" {
			fmt.Fprintf(&short, ": %s", description)
			fmt.Fprintf(&complete, ": %s", description)
		}
		short.WriteString("\n")
		complete.WriteString("\n")
		if text := cleanMarkdown(page.Text, 12000); text != "" {
			fmt.Fprintf(&complete, "\n### %s\n\n%s\n", title, text)
		}
		if short.Len() > maxLLMSBytes || complete.Len() > maxLLMSBytes {
			return nil, nil, fmt.Errorf("discovery output exceeds size limit")
		}
	}
	if count == 0 {
		short.WriteString("No published pages are available.\n")
		complete.WriteString("No published pages are available.\n")
	}
	return []byte(short.String()), []byte(complete.String()), nil
}

type DiscoveryRoute struct {
	Pattern string
	Handler http.Handler
}

func DiscoveryRoutes(provider DiscoveryProvider, cache *DiscoveryCache) []DiscoveryRoute {
	if provider == nil {
		return nil
	}
	if cache == nil {
		cache = &DiscoveryCache{}
	}
	handle := func(full bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data, err := cache.Get(r.Context(), provider, full)
			if err != nil {
				http.Error(w, "discovery unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=60")
			_, _ = w.Write(data)
		}
	}
	return []DiscoveryRoute{
		{Pattern: "GET /llms.txt", Handler: handle(false)},
		{Pattern: "GET /llms-full.txt", Handler: handle(true)},
	}
}

func cleanMarkdown(value string, limit int) string {
	value = plainText(value)
	value = strings.Join(strings.Fields(value), " ")
	value = strings.ReplaceAll(value, "[", "\\[")
	value = strings.ReplaceAll(value, "]", "\\]")
	value = strings.ReplaceAll(value, "(", "\\(")
	value = strings.ReplaceAll(value, ")", "\\)")
	if len(value) > limit {
		value = value[:limit]
	}
	return strings.ToValidUTF8(value, "")
}

func plainText(value string) string {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(value))
	var output strings.Builder
	skip := ""
	for {
		typ := tokenizer.Next()
		switch typ {
		case xhtml.ErrorToken:
			return output.String()
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "script" || token.Data == "style" || token.Data == "svg" {
				skip = token.Data
			}
		case xhtml.EndTagToken:
			token := tokenizer.Token()
			if token.Data == skip {
				skip = ""
			}
		case xhtml.TextToken:
			if skip == "" {
				output.Write(tokenizer.Text())
				output.WriteByte(' ')
			}
		}
	}
}

func cleanPath(value string) string {
	value = strings.TrimSpace(value)
	parsed, err := url.ParseRequestURI(value)
	if err != nil || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(value, "\\\r\n") {
		return ""
	}
	path := parsed.EscapedPath()
	path = strings.NewReplacer("(", "%28", ")", "%29", "[", "%5B", "]", "%5D").Replace(path)
	return path
}

func excluded(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix = strings.TrimSpace(prefix); prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
