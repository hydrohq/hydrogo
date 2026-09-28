package media

import (
	"embed"
	"net/http"
)

//go:embed assets/uploader.js
var assets embed.FS

func uploaderAsset(w http.ResponseWriter, _ *http.Request) {
	data, err := assets.ReadFile("assets/uploader.js")
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(data)
}
