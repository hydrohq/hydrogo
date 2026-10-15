package transport

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
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
}

// TestMemberAuthRendersAppearanceCSS closes the gap recorded in
// docs/decisions/0004-theme-appearance-css.md: modules/news already gained
// the admin's Appearance brand tokens; modules/members did not.
func TestMemberAuthRendersAppearanceCSS(t *testing.T) {
	bundle, err := v1.LoadTheme(filepath.Join("..", "..", "..", "themes"), "newsroom")
	if err != nil {
		t.Fatal(err)
	}
	h := Handler{Theme: &bundle, PublicContactSettings: func(context.Context) ([]site.Setting, error) { return nil, nil }}

	response := httptest.NewRecorder()
	response.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'")
	h.loginPage(response, httptest.NewRequest("GET", "/member/login", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "<style>:root{--brand:") {
		t.Fatalf("member login page missing appearance style: %s", body)
	}
	if policy := response.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "sha256-") {
		t.Fatalf("CSP was not extended for the appearance style: %s", policy)
	}
}
