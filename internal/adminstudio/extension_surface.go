package adminstudio

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// ServeExtensionSurface renders the isolated host shell for one validated
// external surface. It intentionally sends only non-secret bootstrap context;
// session cookies and bearer tokens never cross into the iframe.
func (r ExtensionRegistry) ServeExtensionSurface(w http.ResponseWriter, req *http.Request, extensionID, surfaceID, locale string, principal ChannelPrincipal) {
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
	// With allow-same-origin enabled, embedding a surface on HydroGo's own
	// origin would let same-origin extension code escape the sandbox. Signed
	// manifests must remain externally hosted; reject a host collision at the
	// request boundary while retaining the manifest's HTTPS validation.
	if req != nil && strings.EqualFold(parsed.Host, req.Host) {
		http.NotFound(w, req)
		return
	}
	if strings.TrimSpace(locale) == "" || len(locale) > 32 || strings.ContainsAny(locale, "\r\n") {
		locale = "en"
	}
	permissions := []string(nil)
	for _, surface := range manifest.Surfaces {
		if surface.ID == surfaceID {
			permissions = append(permissions, surface.Permissions...)
			break
		}
	}
	channel, err := r.channels.Issue(principal, manifest.ID, surfaceID, parsed.Scheme+"://"+parsed.Host, permissions)
	if err != nil {
		http.Error(w, "extension unavailable", http.StatusServiceUnavailable)
		return
	}
	initPayload := extensionv1.AdminChannelInitPayload{ExtensionID: manifest.ID, SurfaceID: surfaceID, ChannelID: channel.ID, Challenge: channel.Challenge}
	if err := initPayload.Validate(); err != nil {
		http.Error(w, "extension unavailable", http.StatusServiceUnavailable)
		return
	}
	init := extensionv1.AdminMessage{Version: extensionv1.AdminMessageVersion, Type: "init"}
	init.Payload, _ = json.Marshal(initPayload)
	messageBytes, _ := json.Marshal(init)
	data := extensionSurfaceData{URL: template.HTMLEscapeString(surfaceURL), Origin: template.HTMLEscapeString(parsed.Scheme + "://" + parsed.Host), Message: template.JSEscapeString(string(messageBytes)), Challenge: template.JSEscapeString(channel.Challenge), ChannelID: template.JSEscapeString(channel.ID), CSRF: template.JSEscapeString(security.CSRFTokenFromContext(req.Context()))}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// The host shell alone may call its same-origin channel lifecycle endpoints.
	// The frame keeps its external origin so exact event.origin validation remains
	// possible; it is still cross-origin from HydroGo and receives no ambient
	// HydroGo cookies or bearer tokens.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-src "+data.Origin+"; script-src 'unsafe-inline'; connect-src 'self'")
	_ = extensionSurfaceTemplate.Execute(w, data)
}

type extensionSurfaceData struct {
	URL       string
	Origin    string
	Message   string
	Challenge string
	ChannelID string
	CSRF      string
}

var extensionSurfaceTemplate = template.Must(template.New("admin-extension-surface").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Admin extension</title></head>
<body style="margin:0;overflow:hidden"><iframe title="Admin extension" src="{{.URL}}" sandbox="allow-scripts allow-same-origin allow-forms allow-popups" referrerpolicy="no-referrer" style="border:0;width:100vw;height:100vh"></iframe>
<script>(function(){'use strict';var origin='{{.Origin}}',challenge='{{.Challenge}}',channelID='{{.ChannelID}}',csrf='{{.CSRF}}',frame=document.querySelector('iframe'),bound=false;function send(message,ports){if(frame&&frame.contentWindow){frame.contentWindow.postMessage(message,origin,ports||[]);}}function bind(){return fetch('/admin/extensions/channels/bind',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':csrf},credentials:'same-origin',body:JSON.stringify({channel_id:channelID,challenge:challenge})}).then(function(response){return response.ok;});}function revoke(){return fetch('/admin/extensions/channels/revoke',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':csrf},credentials:'same-origin',keepalive:true,body:JSON.stringify({channel_id:channelID})});}window.addEventListener('message',function(event){if(bound||event.origin!==origin||!frame||event.source!==frame.contentWindow||!event.data||event.data.version!==1||event.data.type!=='ready'||!event.data.payload||event.data.payload.challenge!==challenge){return;}bind().then(function(ok){if(!ok||bound){return;}bound=true;var channel=new MessageChannel();channel.port1.start();send({version:1,type:'bound',payload:{channel_id:channelID}},[channel.port2]);});});frame.addEventListener('load',function(){send(JSON.parse('{{.Message}}'));});window.addEventListener('beforeunload',function(){if(bound){revoke();send({version:1,type:'revoke',payload:{channel_id:channelID}});}});}());</script></body></html>`))
