package designsystem

import (
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed assets/designsystem.css assets/widgets.js assets/article-reading.css assets/article-reading.js assets/feedback.js
var assetFS embed.FS

var publicAssetNames = map[string]bool{
	"designsystem.css":    true,
	"widgets.js":          true,
	"article-reading.css": true,
	"article-reading.js":  true,
	"feedback.js":         true,
}

// ServeAsset exposes only the release-owned design system assets.
func ServeAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.PathValue("path"), "/")
	if !publicAssetNames[name] {
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
