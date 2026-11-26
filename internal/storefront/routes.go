package storefront

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RoutesManifestFile is the optional file a single-page storefront ships next
// to its index.html so the server can answer unknown URLs with a real 404.
//
// Without it a SPA fallback serves index.html with status 200 for every path
// (/.env, /anything), which search engines report as "soft 404". With it the
// page body is still index.html -- the SPA renders its own "not found" view --
// but the status is 404 and the page is marked noindex.
//
//	{
//	  "exact":    ["/", "/urunler", "/sepet"],
//	  "prefixes": ["/siparis/"],
//	  "dynamic":  [{"prefix": "/urun/", "collection": "products", "field": "slug"}]
//	}
//
// "exact" paths and "prefixes" (any non-empty remainder) are known as written.
// A "dynamic" route is known only when its single path segment is the value of
// field on a published record of collection (checked through SlugLookup).
const RoutesManifestFile = "routes.json"

// SlugLookup reports whether a published record of collection has field==value.
// Errors fail open: a storefront must never 404 real pages because a lookup broke.
type SlugLookup func(ctx context.Context, collection, field, value string) (bool, error)

type routeManifest struct {
	Exact    []string       `json:"exact"`
	Prefixes []string       `json:"prefixes"`
	Dynamic  []dynamicRoute `json:"dynamic"`
}

type dynamicRoute struct {
	Prefix     string `json:"prefix"`
	Collection string `json:"collection"`
	Field      string `json:"field"`
}

const (
	manifestRecheck = 10 * time.Second
	lookupTTL       = 30 * time.Second
	lookupCacheCap  = 2048
)

type lookupEntry struct {
	ok      bool
	expires time.Time
}

// routeChecker loads routes.json (re-reading it when it changes) and decides
// whether a request path is a real page of the storefront.
type routeChecker struct {
	dir string

	mu        sync.Mutex
	manifest  *routeManifest
	modTime   time.Time
	checkedAt time.Time

	lookup SlugLookup
	cache  map[string]lookupEntry
}

func newRouteChecker(dir string) *routeChecker {
	return &routeChecker{dir: dir, cache: make(map[string]lookupEntry)}
}

func (c *routeChecker) setLookup(fn SlugLookup) {
	c.mu.Lock()
	c.lookup = fn
	c.mu.Unlock()
}

func (c *routeChecker) load() *routeManifest {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if now.Sub(c.checkedAt) < manifestRecheck {
		return c.manifest
	}
	c.checkedAt = now
	file := filepath.Join(c.dir, RoutesManifestFile)
	stat, err := os.Stat(file)
	if err != nil || stat.IsDir() {
		c.manifest, c.modTime = nil, time.Time{}
		return nil
	}
	if c.manifest != nil && stat.ModTime().Equal(c.modTime) {
		return c.manifest
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return c.manifest
	}
	var parsed routeManifest
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// A broken manifest must not take the site down: behave as if absent.
		c.manifest, c.modTime = nil, stat.ModTime()
		return nil
	}
	c.manifest, c.modTime = &parsed, stat.ModTime()
	return c.manifest
}

// known reports whether urlPath is a page the storefront can render. It is
// true whenever there is nothing to judge by (no manifest, lookup failures).
func (c *routeChecker) known(ctx context.Context, urlPath string) bool {
	manifest := c.load()
	if manifest == nil {
		return true
	}
	p := urlPath
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	for _, exact := range manifest.Exact {
		if p == exact {
			return true
		}
	}
	for _, prefix := range manifest.Prefixes {
		if prefix != "" && strings.HasPrefix(p, prefix) && len(p) > len(prefix) {
			return true
		}
	}
	for _, route := range manifest.Dynamic {
		if route.Prefix == "" || route.Collection == "" || route.Field == "" || !strings.HasPrefix(p, route.Prefix) {
			continue
		}
		value := strings.TrimPrefix(p, route.Prefix)
		if value == "" || strings.Contains(value, "/") {
			continue
		}
		if c.slugExists(ctx, route, value) {
			return true
		}
	}
	return false
}

func (c *routeChecker) slugExists(ctx context.Context, route dynamicRoute, value string) bool {
	key := route.Collection + "\x00" + route.Field + "\x00" + value
	now := time.Now()

	c.mu.Lock()
	lookup := c.lookup
	if entry, ok := c.cache[key]; ok && now.Before(entry.expires) {
		c.mu.Unlock()
		return entry.ok
	}
	c.mu.Unlock()

	if lookup == nil {
		return true // cannot verify: fail open
	}
	ok, err := lookup(ctx, route.Collection, route.Field, value)
	if err != nil {
		return true
	}

	c.mu.Lock()
	if len(c.cache) >= lookupCacheCap {
		c.cache = make(map[string]lookupEntry, lookupCacheCap)
	}
	c.cache[key] = lookupEntry{ok: ok, expires: now.Add(lookupTTL)}
	c.mu.Unlock()
	return ok
}

// statusOverrideWriter turns the 200 that http.ServeContent writes into code.
type statusOverrideWriter struct {
	http.ResponseWriter
	code int
	sent bool
}

func (w *statusOverrideWriter) WriteHeader(code int) {
	if !w.sent {
		w.sent = true
		if code == http.StatusOK {
			code = w.code
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusOverrideWriter) Write(b []byte) (int, error) {
	if !w.sent {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
