package transport

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// allowAppearanceStyle adds the hash of the injected <style>{{AppearanceCSS}}</style>
// element (modules/site Appearance, machine-generated from validated hex
// colors and an allowlisted font) to the response's strict CSP so the
// browser does not block it. Mirrors modules/news/transport's helper of the
// same name; only an existing policy is extended, and never one that
// already allows inline styles.
func allowAppearanceStyle(header http.Header, css template.CSS) {
	if css == "" {
		return
	}
	policy := header.Get("Content-Security-Policy")
	if policy == "" || styleSrcAllowsInline(policy) {
		return
	}
	node := &html.Node{Type: html.ElementNode, DataAtom: atom.Style, Data: atom.Style.String()}
	node.AppendChild(&html.Node{Type: html.TextNode, Data: string(css)})
	var rendered bytes.Buffer
	if err := html.Render(&rendered, node); err != nil {
		return
	}
	directives := security.InlineCSPDirectives(rendered.Bytes())
	if directives == "" {
		return
	}
	header.Set("Content-Security-Policy", security.MergeCSP(policy, directives))
}

func styleSrcAllowsInline(policy string) bool {
	for _, directive := range strings.Split(policy, ";") {
		fields := strings.Fields(directive)
		if len(fields) > 0 && fields[0] == "style-src" {
			for _, source := range fields[1:] {
				if source == "'unsafe-inline'" {
					return true
				}
			}
		}
	}
	return false
}
