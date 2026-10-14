package designsystem

import (
	"strings"
	"testing"
)

func TestEditPresenceUsesSafeDOMText(t *testing.T) {
	markup := render(t, EditPresence("page-1", "/admin/pages/page-1/presence", "csrf-token"))
	for _, value := range []string{`data-endpoint="/admin/pages/page-1/presence"`, `data-csrf="csrf-token"`, "textContent=editor.name", "active_seconds"} {
		if !strings.Contains(markup, value) {
			t.Errorf("presence component missing %q", value)
		}
	}
}
