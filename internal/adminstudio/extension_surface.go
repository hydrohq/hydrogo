package adminstudio

import (
	_ "embed"
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
func (r ExtensionRegistry) ServeExtensionSurface(w http.ResponseWriter, req *http.Request, extensionID, surfaceID, locale string, principal ChannelPrincipal, authorizer func(string) bool) {
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
			for _, p := range surface.Permissions {

				if authorizer != nil && authorizer(p) {

					permissions = append(permissions, p)

				}

			}
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

// extensionSurfaceHTML is the standalone host document that frames an
// extension surface. It is its own document with its own CSP (not an admin
// shell page), so it stays an html/template whose inline script gets
// html/template's JavaScript-context escaping.
//
//go:embed extension_surface.html
var extensionSurfaceHTML string

var extensionSurfaceTemplate = template.Must(template.New("admin-extension-surface").Parse(extensionSurfaceHTML))
