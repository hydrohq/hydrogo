package server

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

const (
	publicHTMLBodyLimit  = 512 << 10
	publicHTMLByteLimit  = 32 << 20
	publicHTMLEntryLimit = 128
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
		key := strings.ToLower(r.Host) + "\x00" + r.URL.Path
		if entry, ok := c.get(key); ok {
			if writeCachedHTML(w, r, entry) {
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		tag := cacheTag(r.URL.Path)
		generation := c.currentGeneration(tag)
		pathGeneration := c.beginPathRender(r.URL.Path)
		defer c.endPathRender(r.URL.Path)
		capture := &publicHTMLResponseWriter{header: w.Header(), limit: c.maxBody, dst: w}
		next.ServeHTTP(capture, r)
		if capture.committed {
			return
		}
		body := capture.body.Bytes()
		cacheCSP := capture.header.Get("Content-Security-Policy")
		if cacheCSP == "" && security.NonceFromContext(r.Context()) != "" {
			cacheCSP = "script-src 'nonce-" + security.NonceFromContext(r.Context()) + "'"
		}
		nonceHeader := capture.header.Clone()
		if cacheCSP != "" {
			nonceHeader.Set("Content-Security-Policy", cacheCSP)
		}
		storedBody, storedCSP, cacheable := normalizePublicHTMLNonce(nonceHeader, body)
		cacheHeader := capture.header.Clone()
		if storedCSP != "" {
			cacheHeader.Set("Content-Security-Policy", storedCSP)
		}
		if !cacheable || !publicHTMLResponseAllowed(capture.status, cacheHeader, storedBody) {
			capture.commit(w)
			return
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
	c.generation["pages"]++
	c.generation["news"]++
	for key, entry := range c.entries {
		c.generation[entry.tag]++
		delete(c.entries, key)
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
	csp := strings.ToLower(header.Get("Content-Security-Policy"))
	cacheControl := strings.ToLower(header.Get("Cache-Control"))
	vary := strings.ToLower(strings.TrimSpace(header.Get("Vary")))
	return status == http.StatusOK && strings.HasPrefix(contentType, "text/html") && len(body) > 0 &&
		len(header.Values("Set-Cookie")) == 0 && !strings.Contains(cacheControl, "private") &&
		!strings.Contains(cacheControl, "no-store") && !strings.Contains(cacheControl, "no-cache") &&
		(vary == "" || vary == "accept-encoding") &&
		(!strings.Contains(csp, "nonce-") || strings.Contains(csp, "__hydro_nonce__")) &&
		(!bytes.Contains(bytes.ToLower(body), []byte("nonce=")) || bytes.Contains(body, []byte("__hydro_nonce__")))
}

func writeCachedHTML(w http.ResponseWriter, r *http.Request, entry publicHTMLCacheEntry) bool {
	body := entry.body
	if entry.csp != "" {
		nonce := security.NonceFromContext(r.Context())
		if nonce == "" {
			nonceBytes := make([]byte, 18)
			if _, err := rand.Read(nonceBytes); err != nil {
				return false
			}
			nonce = base64.RawStdEncoding.EncodeToString(nonceBytes)
			w.Header().Set("Content-Security-Policy", strings.ReplaceAll(entry.csp, "__hydro_nonce__", nonce))
		}
		body = bytes.ReplaceAll(body, []byte("__hydro_nonce__"), []byte(nonce))
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

var (
	cspNoncePattern  = regexp.MustCompile(`nonce-([A-Za-z0-9+/_=-]+)`)
	bodyNoncePattern = regexp.MustCompile(`(?i)(\bnonce\s*=\s*)(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
)

func normalizePublicHTMLNonce(header http.Header, body []byte) ([]byte, string, bool) {
	csp := header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "nonce-") {
		return body, "", !bytes.Contains(bytes.ToLower(body), []byte("nonce="))
	}
	matches := cspNoncePattern.FindAllStringSubmatch(csp, -1)
	if len(matches) == 0 {
		return nil, "", false
	}
	nonce := matches[0][1]
	for _, match := range matches[1:] {
		if match[1] != nonce {
			return nil, "", false
		}
	}
	for _, match := range bodyNoncePattern.FindAllSubmatch(body, -1) {
		value := match[2]
		if value == nil {
			value = match[3]
		}
		if value == nil {
			value = match[4]
		}
		if string(value) != nonce {
			return nil, "", false
		}
	}
	normalizedCSP := strings.ReplaceAll(csp, "nonce-"+nonce, "nonce-__hydro_nonce__")
	normalizedBody := bodyNoncePattern.ReplaceAllFunc(body, func(match []byte) []byte {
		parts := bodyNoncePattern.FindSubmatch(match)
		return append(append(bytes.Clone(parts[1]), '"'), append([]byte("__hydro_nonce__"), '"')...)
	})
	return normalizedBody, normalizedCSP, true
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
