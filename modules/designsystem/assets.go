package designsystem

import (
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed assets/designsystem.css assets/widgets.js
var assetFS embed.FS

// ServeAsset exposes only the release-owned design system assets.
func ServeAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.PathValue("path"), "/")
	if name != "designsystem.css" && name != "widgets.js" {
		http.NotFound(w, r)
		return
	}
	data, err := assetFS.ReadFile("assets/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(name)))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, name, time.Time{}, strings.NewReader(string(data)))
}
