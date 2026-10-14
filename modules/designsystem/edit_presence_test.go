package designsystem

import (
	"strings"
	"testing"
)

func TestEditPresenceLoadsItsScriptFromAdminAssets(t *testing.T) {
	markup := render(t, EditPresence("page-1", "/admin/pages/page-1/presence", "csrf-token"))
	for _, value := range []string{`data-endpoint="/admin/pages/page-1/presence"`, `data-csrf="csrf-token"`, `src="/admin-assets/js/edit-presence.js"`} {
		if !strings.Contains(markup, value) {
			t.Errorf("presence component missing %q", value)
		}
	}
	if strings.Contains(markup, "<script>") {
		t.Error("presence component renders an inline script the admin CSP blocks")
	}
}
