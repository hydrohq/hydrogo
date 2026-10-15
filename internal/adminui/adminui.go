package adminui

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/router"
)

//go:embed all:dist
var distFS embed.FS

// Routes returns router routes for serving the embedded Svelte Admin SPA.
func Routes() []router.Route {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		indexHTML = []byte("Admin SPA unavailable")
	}

	indexSum := sha256.Sum256(indexHTML)
	indexETag := `"` + hex.EncodeToString(indexSum[:8]) + `"`

	fileServer := http.FileServer(http.FS(sub))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/admin")
		reqPath = strings.TrimPrefix(reqPath, "/")

		if reqPath == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("ETag", indexETag)
			if match := r.Header.Get("If-None-Match"); match == indexETag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(indexHTML)
			return
		}

		cleanPath := path.Clean(reqPath)
		f, err := sub.Open(cleanPath)
		if err == nil {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !stat.IsDir() {
				if strings.HasPrefix(cleanPath, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				http.StripPrefix("/admin/", fileServer).ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: return index.html for client-side navigation
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", indexETag)
		if match := r.Header.Get("If-None-Match"); match == indexETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexHTML)
	})

	return []router.Route{
		{Pattern: "GET /admin", Access: router.Public, Handler: handler},
		{Pattern: "GET /admin/{path...}", Access: router.Public, Handler: handler},
	}
}
