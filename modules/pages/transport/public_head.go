package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// publicPageView is the theme-less fallback document. Every string except
// Locale, Direction and Title is already-rendered, trusted markup.
type publicPageView struct {
	Locale, Direction, Title               string
	Head, Search, EditLink, Tools, Article string
	Blocks, Related, Consent, Support      string
}

// seoAlternate is one published translation of the page.
type seoAlternate struct{ Locale, Href string }

// seoHeadView is the vetted data for a public page's head additions. URL
// fields are validated (local font and favicon paths, the configured origin)
// before they get here; empty fields are omitted.
type seoHeadView struct {
	AppearanceCSS, FontCSS, FaviconURL string
	BusinessJSONLD                     json.RawMessage
	Social                             bool
	Alternates                         []seoAlternate
	Canonical, Description             string
	OGTitle, OGDescription, OGImage    string
}

// inlineStyle renders a style element around machine-generated CSS (the
// validated appearance tokens). templ does not interpolate inside style
// elements, so the node is built and serialised with x/net/html, which
// writes style text raw, exactly as a browser parses it.
func inlineStyle(css string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		node := &html.Node{Type: html.ElementNode, DataAtom: atom.Style, Data: atom.Style.String()}
		node.AppendChild(&html.Node{Type: html.TextNode, Data: css})
		return html.Render(w, node)
	})
}

// allowHeadStyles adds the hashes of the head's inline style elements (the
// design-system critical CSS and the appearance tokens, both machine-owned)
// to the response policy so a strict style-src does not block them. Only an
// existing policy is extended, and never one that already allows inline
// styles: adding a hash would switch 'unsafe-inline' off.
func allowHeadStyles(header http.Header, head string) {
	policy := header.Get("Content-Security-Policy")
	if policy == "" || styleSrcAllowsInline(policy) {
		return
	}
	_, styles := security.InlineCSPHashes([]byte(head))
	if len(styles) == 0 {
		return
	}
	sources := make([]string, len(styles))
	for i, hash := range styles {
		sources[i] = "'sha256-" + hash + "'"
	}
	header.Set("Content-Security-Policy", security.MergeCSP(policy, "style-src "+strings.Join(sources, " ")))
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
