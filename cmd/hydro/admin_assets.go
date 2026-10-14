package main

import (
	"io/fs"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/router"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

// adminAssetsRoute serves the admin's own embedded CSS/JS (static/admin) at
// /admin-assets/. Responses are revalidated on every load so a new binary's
// assets are picked up without cache-busting query strings.
func adminAssetsRoute() router.Route {
	assets, err := fs.Sub(staticassets.FS, "admin")
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	files := http.StripPrefix("/admin-assets/", http.FileServer(http.FS(assets)))
	return router.Route{Pattern: "GET /admin-assets/{path...}", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})}
}
