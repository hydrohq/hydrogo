package adminstudio

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

// ServeExtensionSurface renders the isolated host shell for one validated
// external surface. It intentionally sends only non-secret bootstrap context;
// session cookies and bearer tokens never cross into the iframe.
func (r ExtensionRegistry) ServeExtensionSurface(w http.ResponseWriter, req *http.Request, extensionID, surfaceID, locale string) {
	manifest, ok := r.Manifest(extensionID)
	if !ok {
		http.NotFound(w, req)
		return
	}
	surfaceURL, ok := manifest.SurfaceURL(surfaceID)
	if !ok {
		http.NotFound(w, req)
		return
	}
	parsed, err := url.Parse(surfaceURL)
	if err != nil {
		http.NotFound(w, req)
		return
	}
	if strings.TrimSpace(locale) == "" || len(locale) > 32 || strings.ContainsAny(locale, "\r\n") {
		locale = "en"
	}
	bootstrap := extensionv1.AdminMessage{Version: extensionv1.AdminMessageVersion, Type: "ready"}
	payload := extensionv1.AdminBootstrapPayload{ExtensionID: manifest.ID, SurfaceID: surfaceID, Locale: locale}
	for _, surface := range manifest.Surfaces {
		if surface.ID == surfaceID {
			payload.Permissions = append([]string(nil), surface.Permissions...)
			break
		}
	}
	payloadBytes, _ := json.Marshal(payload)
	bootstrap.Payload = payloadBytes
	messageBytes, _ := json.Marshal(bootstrap)
	data := extensionSurfaceData{URL: template.HTMLEscapeString(surfaceURL), Origin: template.HTMLEscapeString(parsed.Scheme + "://" + parsed.Host), Message: template.JSEscapeString(string(messageBytes))}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-src "+data.Origin+"; script-src 'unsafe-inline'")
	_ = extensionSurfaceTemplate.Execute(w, data)
}

type extensionSurfaceData struct {
	URL     string
	Origin  string
	Message string
}

var extensionSurfaceTemplate = template.Must(template.New("admin-extension-surface").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Admin extension</title></head>
<body style="margin:0;overflow:hidden"><iframe title="Admin extension" src="{{.URL}}" sandbox="allow-scripts allow-forms allow-popups" referrerpolicy="no-referrer" style="border:0;width:100vw;height:100vh"></iframe>
<script>window.addEventListener('load',function(){var frame=document.querySelector('iframe');if(frame&&frame.contentWindow){frame.contentWindow.postMessage(JSON.parse('{{.Message}}'),'{{.Origin}}')}})</script></body></html>`))
