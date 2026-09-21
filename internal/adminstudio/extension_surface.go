package adminstudio

import (
	"crypto/rand"
	"encoding/hex"
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
	challenge, err := newExtensionChallenge()
	if err != nil {
		http.Error(w, "extension unavailable", http.StatusServiceUnavailable)
		return
	}
	initPayload := extensionv1.AdminChannelInitPayload{ExtensionID: manifest.ID, SurfaceID: surfaceID, Challenge: challenge}
	if err := initPayload.Validate(); err != nil {
		http.Error(w, "extension unavailable", http.StatusServiceUnavailable)
		return
	}
	init := extensionv1.AdminMessage{Version: extensionv1.AdminMessageVersion, Type: "init"}
	init.Payload, _ = json.Marshal(initPayload)
	messageBytes, _ := json.Marshal(init)
	data := extensionSurfaceData{URL: template.HTMLEscapeString(surfaceURL), Origin: template.HTMLEscapeString(parsed.Scheme + "://" + parsed.Host), Message: template.JSEscapeString(string(messageBytes)), Challenge: template.JSEscapeString(challenge)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-src "+data.Origin+"; script-src 'unsafe-inline'")
	_ = extensionSurfaceTemplate.Execute(w, data)
}

type extensionSurfaceData struct {
	URL       string
	Origin    string
	Message   string
	Challenge string
}

var extensionSurfaceTemplate = template.Must(template.New("admin-extension-surface").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Admin extension</title></head>
<body style="margin:0;overflow:hidden"><iframe title="Admin extension" src="{{.URL}}" sandbox="allow-scripts allow-forms allow-popups" referrerpolicy="no-referrer" style="border:0;width:100vw;height:100vh"></iframe>
<script>(function(){'use strict';var origin='{{.Origin}}',challenge='{{.Challenge}}',frame=document.querySelector('iframe'),bound=false;function send(message,ports){if(frame&&frame.contentWindow){frame.contentWindow.postMessage(message,origin,ports||[]);}}window.addEventListener('message',function(event){if(bound||event.origin!==origin||!frame||event.source!==frame.contentWindow||!event.data||event.data.version!==1||event.data.type!=='ready'||!event.data.payload||event.data.payload.challenge!==challenge){return;}bound=true;var channel=new MessageChannel();channel.port1.start();send({version:1,type:'bound',payload:{challenge:challenge}},[channel.port2]);});frame.addEventListener('load',function(){send(JSON.parse('{{.Message}}'));});window.addEventListener('beforeunload',function(){if(bound){send({version:1,type:'revoke',payload:{challenge:challenge}});}});}());</script></body></html>`))

func newExtensionChallenge() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
