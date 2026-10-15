package transport

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func TestAllowAppearanceStyleHashesOnlyMachineOwnedStyle(t *testing.T) {
	css := template.CSS(":root{--brand:#155EEF}")
	_, styles := security.InlineCSPHashes([]byte("<style>" + string(css) + "</style>"))
	if len(styles) != 1 {
		t.Fatalf("expected one inline style hash, got %v", styles)
	}

	strict := http.Header{}
	strict.Set("Content-Security-Policy", "default-src 'self'; style-src 'self'")
	allowAppearanceStyle(strict, css)
	if policy := strict.Get("Content-Security-Policy"); !strings.Contains(policy, "'sha256-"+styles[0]+"'") {
		t.Fatalf("strict policy missing the appearance hash: %s", policy)
	}

	relaxed := http.Header{}
	relaxed.Set("Content-Security-Policy", "style-src 'self' 'unsafe-inline'")
	allowAppearanceStyle(relaxed, css)
	if policy := relaxed.Get("Content-Security-Policy"); strings.Contains(policy, "sha256-") {
		t.Fatalf("a hash would disable 'unsafe-inline': %s", policy)
	}

	unset := http.Header{}
	allowAppearanceStyle(unset, css)
	if policy := unset.Get("Content-Security-Policy"); policy != "" {
		t.Fatalf("an unset policy must stay unset, got %s", policy)
	}

	empty := http.Header{}
	empty.Set("Content-Security-Policy", "default-src 'self'; style-src 'self'")
	allowAppearanceStyle(empty, "")
	if policy := empty.Get("Content-Security-Policy"); strings.Contains(policy, "sha256-") {
		t.Fatalf("empty appearance CSS must not add a hash: %s", policy)
	}
}

// TestPublicNewsRendersAppearanceCSS closes the gap found while auditing the
// newsroom theme: modules/pages already injects the admin's Appearance
// brand tokens into the page head, but modules/news did not, so the
// configured brand color never reached the site's actual homepage
// (news-list). It must render the tokens and extend a strict CSP to allow
// the exact inline <style> it emits.
func TestPublicNewsRendersAppearanceCSS(t *testing.T) {
	h := publicNewsHandler(t, loadShippedTheme(t, "newsroom"))
	h.PublicContactSettings = func(context.Context) ([]site.Setting, error) { return nil, nil }

	request := httptest.NewRequest("GET", "/public/news/tr/html", nil)
	request.SetPathValue("locale", "tr")
	response := httptest.NewRecorder()
	response.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'")
	h.publicListHTML(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "<style>:root{--brand:") {
		t.Fatalf("news list missing appearance style: %s", body)
	}
	if policy := response.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "sha256-") {
		t.Fatalf("CSP was not extended for the appearance style: %s", policy)
	}
}
