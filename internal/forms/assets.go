package forms

import (
	"embed"
	"net/http"
	"path"
	"strings"
)

//go:embed assets/forms.css assets/forms.js
var assets embed.FS

func ServeAsset(w http.ResponseWriter, r *http.Request) {
	name := path.Base(strings.TrimSpace(r.PathValue("path")))
	if name != "forms.css" && name != "forms.js" {
		http.NotFound(w, r)
		return
	}
	content, err := assets.ReadFile("assets/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if name == "forms.css" {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(content)
}
