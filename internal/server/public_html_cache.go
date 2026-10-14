package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// These are outer safety ceilings, not the effective size for any profile.
// ADR-0334 scales the actual per-profile budget (Compact/Standard/Distributed)
// within these bounds via capacity.Budget; NewPublicHTMLCache clamps any
// caller-supplied option down to whichever of these is smallest.
const (
	publicHTMLBodyLimit  = 2 << 20
	publicHTMLByteLimit  = 512 << 20
	publicHTMLEntryLimit = 4096
)

// PublicHTMLCacheOptions bounds the in-process cache for anonymous public
// page and news HTML responses.
type PublicHTMLCacheOptions struct {
	MaxEntries int
	MaxBytes   int64
	MaxBody    int64
	TTL        time.Duration
}

type publicHTMLCacheEntry struct {
	path            string
	tag             string
	body            []byte
	contentType     string
	contentEncoding string
	csp             string
	vary            string
	etag            string
	expiresAt       time.Time
	generation      uint64
}

type publicHTMLFlight struct {
	generation uint64
	active     int
}

// PublicHTMLCache stores only complete, anonymous, public GET HTML responses.
// Its limits prevent cache growth from being proportional to site traffic.
type PublicHTMLCache struct {
	mu         sync.Mutex
	entries    map[string]publicHTMLCacheEntry
	generation map[string]uint64
	paths      map[string]*publicHTMLFlight
	maxEntries int
	maxBytes   int64
	maxBody    int64
	ttl        time.Duration
	bytes      int64
	render     singleflight.Group
}

// publicHTMLRenderOutcome is the value shared by singleflight across every
// request that arrived while one render for the same key was already in
// flight (ADR-0333 note in ADR-0329: coalesces concurrent cache misses).
// Only the single goroutine whose closure singleflight actually runs writes
// a real HTTP response inside that closure; every other caller must render
// its own response afterward, so nothing observable (headers, CSP nonce,
// body) is ever shared between two distinct client responses.
type publicHTMLRenderOutcome struct {
	cacheable bool
	entry     publicHTMLCacheEntry
}

func NewPublicHTMLCache(opts PublicHTMLCacheOptions) *PublicHTMLCache {
	if opts.MaxEntries <= 0 || opts.MaxEntries > publicHTMLEntryLimit {
		opts.MaxEntries = publicHTMLEntryLimit
	}
	if opts.MaxBytes <= 0 || opts.MaxBytes > publicHTMLByteLimit {
		opts.MaxBytes = publicHTMLByteLimit
	}
	if opts.MaxBody <= 0 || opts.MaxBody > publicHTMLBodyLimit {
		opts.MaxBody = publicHTMLBodyLimit
	}
	if opts.TTL <= 0 || opts.TTL > time.Minute {
		opts.TTL = 30 * time.Second
	}
	return &PublicHTMLCache{
		entries: make(map[string]publicHTMLCacheEntry), generation: make(map[string]uint64), paths: make(map[string]*publicHTMLFlight),
		maxEntries: opts.MaxEntries, maxBytes: opts.MaxBytes, maxBody: opts.MaxBody, ttl: opts.TTL,
	}
}

// Middleware caches allowlisted public page/news response routes. Cookies,
// authorization, queries, HTMX fragments, errors, and nonce-bearing HTML bypass it.
func (c *PublicHTMLCache) Middleware(next http.Handler) http.Handler {
	if c == nil || next == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !publicHTMLRequestAllowed(r) {
			next.ServeHTTP(w, r)
			return
		}
		key := strings.ToLower(r.Host) + "\x00" + negotiatedEncoding(r.Header.Get("Accept-Encoding")) + "\x00" + r.URL.Path
		if entry, ok := c.get(key); ok {
			if writeCachedHTML(w, r, entry) {
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		// Coalesce concurrent misses on the same key: only the caller whose
		// closure singleflight actually runs performs a render, using its own
		// w/r end to end exactly as before. Every other concurrent caller for
		// the same key waits here, then serves its own response from the
		// resulting cache entry (never from the leader's raw captured bytes),
		// so per-response CSP nonces are still generated fresh for each
		// client and nothing about one client's response is shared with
		// another's.
		executedHere := false
		outcomeAny, _, _ := c.render.Do(key, func() (any, error) {
			executedHere = true
			tag := cacheTag(r.URL.Path)
			generation := c.currentGeneration(tag)
			pathGeneration := c.beginPathRender(r.URL.Path)
			defer c.endPathRender(r.URL.Path)
			capture := &publicHTMLResponseWriter{header: w.Header(), limit: c.maxBody, dst: w}
			next.ServeHTTP(capture, r)
			if capture.committed {
				return publicHTMLRenderOutcome{}, nil
			}
			body := capture.body.Bytes()
			storedBody := body
			storedCSP := capture.header.Get("Content-Security-Policy")
			cacheable := true
			cacheHeader := capture.header.Clone()
			if !cacheable || !publicHTMLResponseAllowed(capture.status, cacheHeader, storedBody) {
				capture.commit(w)
				return publicHTMLRenderOutcome{}, nil
			}
			hash := sha256.Sum256(storedBody)
			etag := `W/"` + hex.EncodeToString(hash[:]) + `"`
			capture.header.Set("ETag", etag)
			capture.header.Set("Cache-Control", "public, max-age="+strconvSeconds(c.ttl))
			entry := publicHTMLCacheEntry{
				path: r.URL.Path, tag: tag, body: bytes.Clone(storedBody), contentType: capture.header.Get("Content-Type"),
				contentEncoding: capture.header.Get("Content-Encoding"), vary: capture.header.Get("Vary"), csp: storedCSP,
				etag: etag, expiresAt: time.Now().Add(c.ttl),
			}
			c.set(key, entry, generation, pathGeneration)
			if !writeCachedHTML(w, r, entry) {
				next.ServeHTTP(w, r)
			}
			return publicHTMLRenderOutcome{cacheable: true, entry: entry}, nil
		})
		if executedHere {
			return
		}
		outcome, _ := outcomeAny.(publicHTMLRenderOutcome)
		if outcome.cacheable && writeCachedHTML(w, r, outcome.entry) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// PurgePath invalidates only cached host variants for a public route.
func (c *PublicHTMLCache) PurgePath(path string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if flight := c.paths[path]; flight != nil {
		flight.generation++
	}
	for key, entry := range c.entries {
		if entry.path == path {
			c.removeLocked(key)
		}
	}
}

// PurgeTag invalidates entries for one public collection and advances its
// generation so an in-flight render from before invalidation cannot refill it.
func (c *PublicHTMLCache) PurgeTag(tag string) {
	if c == nil || (tag != "pages" && tag != "news") {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation[tag]++
	for key, entry := range c.entries {
		if entry.tag == tag {
			c.removeLocked(key)
		}
	}
}

// Clear invalidates every cached response and any render currently in flight.
func (c *PublicHTMLCache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for tag := range c.generation {
		c.generation[tag]++
	}
	for key, entry := range c.entries {
		c.generation[entry.tag]++
		c.removeLocked(key)
	}
}

func (c *PublicHTMLCache) get(key string) (publicHTMLCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return publicHTMLCacheEntry{}, false
	}
	if time.Now().After(entry.expiresAt) || entry.generation != c.generation[entry.tag] {
		c.removeLocked(key)
		return publicHTMLCacheEntry{}, false
	}
	return entry, true
}

func (c *PublicHTMLCache) currentGeneration(tag string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation[tag]
}

func (c *PublicHTMLCache) beginPathRender(path string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	flight := c.paths[path]
	if flight == nil {
		flight = &publicHTMLFlight{}
		c.paths[path] = flight
	}
	flight.active++
	return flight.generation
}

func (c *PublicHTMLCache) endPathRender(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if flight := c.paths[path]; flight != nil {
		flight.active--
		if flight.active == 0 {
			delete(c.paths, path)
		}
	}
}

func (c *PublicHTMLCache) set(key string, entry publicHTMLCacheEntry, generation, pathGeneration uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	flight := c.paths[entry.path]
	currentPathGeneration := uint64(0)
	if flight != nil {
		currentPathGeneration = flight.generation
	}
	if generation != c.generation[entry.tag] || pathGeneration != currentPathGeneration {
		return
	}
	entry.generation = generation
	if _, ok := c.entries[key]; ok {
		c.removeLocked(key)
	}
	for len(c.entries) >= c.maxEntries || c.bytes+int64(len(entry.body)) > c.maxBytes {
		oldestKey := ""
		var oldest time.Time
		for candidate, current := range c.entries {
			if oldestKey == "" || current.expiresAt.Before(oldest) {
				oldestKey, oldest = candidate, current.expiresAt
			}
		}
		if oldestKey == "" {
			return
		}
		c.removeLocked(oldestKey)
	}
	c.entries[key] = entry
	c.bytes += int64(len(entry.body))
}

func (c *PublicHTMLCache) removeLocked(key string) {
	if entry, ok := c.entries[key]; ok {
		c.bytes -= int64(len(entry.body))
		delete(c.entries, key)
	}
}

type publicHTMLResponseWriter struct {
	header    http.Header
	body      bytes.Buffer
	status    int
	limit     int64
	dst       http.ResponseWriter
	committed bool
}

func (w *publicHTMLResponseWriter) Header() http.Header { return w.header }

func (w *publicHTMLResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *publicHTMLResponseWriter) Write(body []byte) (int, error) {
	if w.committed {
		return w.dst.Write(body)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if int64(w.body.Len()+len(body)) > w.limit {
		w.commit(w.dst)
		return w.dst.Write(body)
	}
	return w.body.Write(body)
}

func (w *publicHTMLResponseWriter) commit(dst http.ResponseWriter) {
	if !w.committed {
		for key, values := range w.header {
			dst.Header()[key] = append([]string(nil), values...)
		}
		status := w.status
		if status == 0 {
			status = http.StatusOK
		}
		dst.WriteHeader(status)
		_, _ = dst.Write(w.body.Bytes())
		w.committed = true
	}
}

func publicHTMLRequestAllowed(r *http.Request) bool {
	return r.Method == http.MethodGet && r.Header.Get("Cookie") == "" && r.Header.Get("Authorization") == "" &&
		!strings.Contains(strings.ToLower(r.Header.Get("Cache-Control")), "no-cache") && staticHTMLRequestAllowed(r, "hydro_session", nil)
}

func publicHTMLResponseAllowed(status int, header http.Header, body []byte) bool {
	contentType := strings.ToLower(header.Get("Content-Type"))
	cacheControl := strings.ToLower(header.Get("Cache-Control"))
	vary := strings.ToLower(strings.TrimSpace(header.Get("Vary")))
	return status == http.StatusOK && strings.HasPrefix(contentType, "text/html") && len(body) > 0 &&
		len(header.Values("Set-Cookie")) == 0 && !strings.Contains(cacheControl, "private") &&
		!strings.Contains(cacheControl, "no-store") && !strings.Contains(cacheControl, "no-cache") &&
		(vary == "" || vary == "accept-encoding") &&
		true
}

func writeCachedHTML(w http.ResponseWriter, r *http.Request, entry publicHTMLCacheEntry) bool {
	body := entry.body
	if entry.csp != "" {
		w.Header().Set("Content-Security-Policy", entry.csp)
	}
	w.Header().Set("Content-Type", entry.contentType)
	if entry.contentEncoding != "" {
		w.Header().Set("Content-Encoding", entry.contentEncoding)
	}
	if entry.vary != "" {
		w.Header().Set("Vary", entry.vary)
	}
	w.Header().Set("ETag", entry.etag)
	maxAge := int(time.Until(entry.expiresAt) / time.Second)
	if maxAge < 0 {
		maxAge = 0
	}
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAge))
	if headerHasETag(r.Header.Get("If-None-Match"), entry.etag) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func cacheTag(path string) string {
	if strings.HasPrefix(path, "/public/pages/") {
		return "pages"
	}
	return "news"
}

func strconvSeconds(ttl time.Duration) string {
	seconds := int(ttl / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}
