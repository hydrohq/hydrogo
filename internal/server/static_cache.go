package server

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// StaticCacheOptions configures the static HTML disk caching middleware.
type StaticCacheOptions struct {
	// BaseDir is the filesystem directory where cached HTML files are written.
	BaseDir string
	// SessionCookie is the cookie name that bypasses static disk caching when present.
	SessionCookie string
	// ExcludedPrefixes contains path prefixes (e.g. "/admin", "/api") that are never cached.
	ExcludedPrefixes []string
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
	excluded := append([]string{"/admin", "/api", "/login", "/healthz", "/readyz", "/csp-report"}, opts.ExcludedPrefixes...)

	var mu sync.Mutex

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache public GET requests without query parameters
			if r.Method != http.MethodGet || len(r.URL.RawQuery) > 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Do not cache authenticated sessions
			if _, err := r.Cookie(sessionCookie); err == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Do not cache excluded prefixes
			urlPath := r.URL.Path
			for _, prefix := range excluded {
				if strings.HasPrefix(urlPath, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Capture response
			recorder := &responseCapture{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				status:         http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			// Only write to cache on successful 200 OK with HTML content
			contentType := recorder.Header().Get("Content-Type")
			if recorder.status == http.StatusOK && strings.HasPrefix(contentType, "text/html") && recorder.body.Len() > 0 {
				mu.Lock()
				defer mu.Unlock()
				_ = writeCachedFile(opts.BaseDir, urlPath, recorder.body.Bytes())
			}
		})
	}
}

type responseCapture struct {
	http.ResponseWriter
	body        *bytes.Buffer
	status      int
	wroteHeader bool
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
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func writeCachedFile(baseDir, urlPath string, content []byte) error {
	cleanPath := filepath.Clean(urlPath)
	var targetFile string
	if cleanPath == "/" || cleanPath == "." {
		targetFile = filepath.Join(baseDir, "index.html")
	} else {
		targetFile = filepath.Join(baseDir, cleanPath+".html")
	}

	targetDir := filepath.Dir(targetFile)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(targetFile, content, 0o644)
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
	cleanPath := filepath.Clean(urlPath)
	var targetFile string
	if cleanPath == "/" || cleanPath == "." {
		targetFile = filepath.Join(baseDir, "index.html")
	} else {
		targetFile = filepath.Join(baseDir, cleanPath+".html")
	}
	return os.Remove(targetFile)
}
