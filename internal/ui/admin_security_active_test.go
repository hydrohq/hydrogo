package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestSecurityAreaHighlightsSidebarActiveItem(t *testing.T) {
	var out bytes.Buffer
	err := RenderAdminArea(&out, AdminAreaData{
		Email: "admin@example.test",
		Nonce: "nonce-sec",
		CSRF:  "csrf-sec",
		Area:  "security",
	})
	if err != nil {
		t.Fatalf("RenderAdminArea(security) error = %v", err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, `class="nav-item is-active" href="/admin/developer"`) {
		t.Errorf("expected developer nav item to have is-active class for security tab, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Güvenlik &amp; Erişim Kontrolü") {
		t.Errorf("expected security title in rendered output")
	}
}
