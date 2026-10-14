package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/router"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

// adminAssetsRoute serves the admin's own embedded CSS/JS (static/admin) at
// /admin-assets/. Every response carries a content-hash ETag and must be
// revalidated, so a new binary's assets are picked up immediately while
// unchanged files cost a 304 instead of a full download.
func adminAssetsRoute() router.Route {
	assets, err := fs.Sub(staticassets.FS, "admin")
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	etags := map[string]string{}
	_ = fs.WalkDir(assets, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(assets, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		etags[path] = `"` + hex.EncodeToString(sum[:8]) + `"`
		return nil
	})
	files := http.StripPrefix("/admin-assets/", http.FileServer(http.FS(assets)))
	return router.Route{Pattern: "GET /admin-assets/{path...}", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if etag, ok := etags[strings.TrimPrefix(r.URL.Path, "/admin-assets/")]; ok {
			// http.FileServer answers If-None-Match with 304 when ETag is set.
			w.Header().Set("ETag", etag)
		}
		files.ServeHTTP(w, r)
	})}
}
