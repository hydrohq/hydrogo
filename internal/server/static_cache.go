package server

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
)

const (
	staticCacheBodyLimit  = 512 << 10
	staticCacheTotalLimit = 32 << 20
	staticCacheFileLimit  = 128
)

var staticCacheSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
var staticCacheHostSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]{0,252}$`)

// StaticCacheOptions configures the static HTML disk caching middleware.
type StaticCacheOptions struct {
	// BaseDir is the filesystem directory where cached HTML files are written.
	BaseDir string
	// SessionCookie is the cookie name that bypasses static disk caching when present.
	SessionCookie string
	// ExcludedPrefixes contains path prefixes (e.g. "/admin", "/api") that are never cached.
	ExcludedPrefixes []string
	MaxBodyBytes     int64
	MaxTotalBytes    int64
	MaxEntries       int
}

// StaticDiskCacheMiddleware writes 200 OK HTML responses for public GET requests
// to disk so that a fronting web server (e.g. Apache with .htaccess) can serve
// subsequent requests directly from disk without invoking CGI or application code.
func StaticDiskCacheMiddleware(opts StaticCacheOptions) func(http.Handler) http.Handler {
	if opts.BaseDir == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	sessionCookie := opts.SessionCookie
	if sessionCookie == "" {
		sessionCookie = "hydro_session"
	}
	if opts.MaxBodyBytes <= 0 || opts.MaxBodyBytes > staticCacheBodyLimit {
		opts.MaxBodyBytes = staticCacheBodyLimit
	}
	if opts.MaxTotalBytes <= 0 || opts.MaxTotalBytes > staticCacheTotalLimit {
		opts.MaxTotalBytes = staticCacheTotalLimit
	}
	if opts.MaxEntries <= 0 || opts.MaxEntries > staticCacheFileLimit {
		opts.MaxEntries = staticCacheFileLimit
	}

	var mu sync.Mutex

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !staticHTMLRequestAllowed(r, sessionCookie, opts.ExcludedPrefixes) {
				next.ServeHTTP(w, r)
				return
			}
			urlPath := r.URL.Path

			// Capture response
			recorder := &responseCapture{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				status:         http.StatusOK,
				initialCookies: append([]string(nil), w.Header().Values("Set-Cookie")...),
			}

			next.ServeHTTP(recorder, r)

			// Only write to cache on successful 200 OK with HTML content
			contentType := recorder.Header().Get("Content-Type")
			baselineCookies := recorder.initialCookies
			body := recorder.body.Bytes()
			if recorder.status == http.StatusOK && !recorder.overflow && int64(len(body)) <= opts.MaxBodyBytes && strings.HasPrefix(contentType, "text/html") && len(body) > 0 && sameCookieHeaders(baselineCookies, recorder.Header().Values("Set-Cookie")) && cacheableHTMLHeaders(recorder.Header()) && !bytes.Contains(bytes.ToLower(body), []byte("nonce=")) {
				mu.Lock()
				defer mu.Unlock()
				_ = writeCachedFileForHostLimit(opts.BaseDir, r.Host, urlPath, body, opts.MaxEntries, opts.MaxTotalBytes)
			}
		})
	}
}

type responseCapture struct {
	http.ResponseWriter
	body           *bytes.Buffer
	status         int
	wroteHeader    bool
	overflow       bool
	initialCookies []string
}

func (r *responseCapture) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
		r.ResponseWriter.WriteHeader(status)
	}
}

func (r *responseCapture) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if !r.overflow && int64(r.body.Len()+len(b)) <= staticCacheBodyLimit {
		_, _ = r.body.Write(b)
	} else {
		r.overflow = true
		r.body.Reset()
	}
	return r.ResponseWriter.Write(b)
}

func writeCachedFile(baseDir, urlPath string, content []byte) error {
	return writeCachedFileForHostLimit(baseDir, "localhost", urlPath, content, staticCacheFileLimit, staticCacheTotalLimit)
}

func writeCachedFileLimit(baseDir, urlPath string, content []byte, maxEntries int, maxTotal int64) error {
	return writeCachedFileForHostLimit(baseDir, "localhost", urlPath, content, maxEntries, maxTotal)
}

func writeCachedFileForHostLimit(baseDir, host, urlPath string, content []byte, maxEntries int, maxTotal int64) error {
	if len(content) > staticCacheBodyLimit {
		return nil
	}
	targetFile, err := staticCachePathForHost(baseDir, host, urlPath)
	if err != nil {
		return err
	}

	targetDir := filepath.Dir(targetFile)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	if err := pathguard.RejectSymlinkComponents(targetDir); err != nil {
		return err
	}
	if err := evictStaticCache(baseDir, targetFile, int64(len(content)), maxEntries, maxTotal); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(targetDir, ".html-cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, targetFile)
}

// PurgeStaticDiskCache removes all cached static HTML files in baseDir.
func PurgeStaticDiskCache(baseDir string) error {
	if baseDir == "" {
		return nil
	}
	return os.RemoveAll(baseDir)
}

// PurgeStaticDiskCachePath removes the cached HTML file for a specific URL path.
func PurgeStaticDiskCachePath(baseDir, urlPath string) error {
	if baseDir == "" {
		return nil
	}
	if !staticCacheURLPath(urlPath) {
		return errors.New("unsupported static HTML cache path")
	}
	hosts, err := os.ReadDir(baseDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var firstErr error
	for _, host := range hosts {
		if !host.IsDir() || !staticCacheHostSegment.MatchString(host.Name()) {
			continue
		}
		targetFile, err := staticCachePathForHost(baseDir, host.Name(), urlPath)
		if err == nil {
			err = os.Remove(targetFile)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func staticHTMLRequestAllowed(r *http.Request, sessionCookie string, excluded []string) bool {
	if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "" || r.Header.Get("HX-Request") != "" || strings.Contains(strings.ToLower(r.Header.Get("Cache-Control")), "no-cache") {
		return false
	}
	for _, prefix := range excluded {
		if strings.HasPrefix(r.URL.Path, prefix) {
			return false
		}
	}
	for _, cookie := range r.Cookies() {
		if cookie.Name != "hydro_csrf" && cookie.Name != "hydro_analytics_consent_id" && cookie.Name != "hydro_analytics_purpose" {
			return false
		}
		if cookie.Name == sessionCookie {
			return false
		}
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 5 && parts[0] == "public" && parts[1] == "pages" && parts[4] == "html" {
		return staticCacheSegment.MatchString(parts[2]) && staticCacheSegment.MatchString(parts[3])
	}
	if len(parts) == 4 && parts[0] == "public" && parts[1] == "news" && parts[3] == "html" {
		return staticCacheSegment.MatchString(parts[2])
	}
	if len(parts) == 5 && parts[0] == "public" && parts[1] == "news" && parts[4] == "html" {
		return staticCacheSegment.MatchString(parts[2]) && staticCacheSegment.MatchString(parts[3])
	}
	return false
}

func staticCachePath(baseDir, urlPath string) (string, error) {
	return staticCachePathForHost(baseDir, "localhost", urlPath)
}

func staticCachePathForHost(baseDir, host, urlPath string) (string, error) {
	if strings.TrimSpace(baseDir) == "" || !staticCacheURLPath(urlPath) {
		return "", errors.New("unsupported static HTML cache path")
	}
	if hostWithPort, _, err := net.SplitHostPort(host); err == nil {
		host = hostWithPort
	} else if strings.Contains(host, ":") {
		return "", errors.New("unsupported static HTML cache host")
	}
	if !staticCacheHostSegment.MatchString(host) {
		return "", errors.New("unsupported static HTML cache host")
	}
	return filepath.Join(baseDir, strings.ToLower(host), filepath.FromSlash(strings.TrimPrefix(urlPath, "/"))+".html"), nil
}

func staticCacheURLPath(urlPath string) bool {
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	if len(parts) == 5 && parts[0] == "public" && parts[1] == "pages" && parts[4] == "html" {
		return staticCacheSegment.MatchString(parts[2]) && staticCacheSegment.MatchString(parts[3])
	}
	if len(parts) == 4 && parts[0] == "public" && parts[1] == "news" && parts[3] == "html" {
		return staticCacheSegment.MatchString(parts[2])
	}
	if len(parts) == 5 && parts[0] == "public" && parts[1] == "news" && parts[4] == "html" {
		return staticCacheSegment.MatchString(parts[2]) && staticCacheSegment.MatchString(parts[3])
	}
	return false
}

func cacheableHTMLHeaders(header http.Header) bool {
	if strings.Contains(strings.ToLower(header.Get("Cache-Control")), "no-store") || strings.Contains(strings.ToLower(header.Get("Cache-Control")), "private") {
		return false
	}
	return header.Get("Vary") == "" || header.Get("Vary") == "Accept-Encoding"
}

func sameCookieHeaders(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func evictStaticCache(baseDir, target string, newBytes int64, maxEntries int, maxTotal int64) error {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	type cacheFile struct {
		path string
		size int64
		mod  time.Time
	}
	files := make([]cacheFile, 0, maxEntries)
	var total int64
	err := filepath.WalkDir(baseDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, cacheFile{path: path, size: info.Size(), mod: info.ModTime()})
		total += info.Size()
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for len(files) >= maxEntries || total+newBytes > maxTotal {
		if len(files) == 0 {
			break
		}
		victim := files[0]
		files = files[1:]
		if filepath.Clean(victim.path) == filepath.Clean(target) {
			continue
		}
		if err := os.Remove(victim.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		total -= victim.size
	}
	return nil
}
