package adminstudio

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderKitBadgesEscapesAndSorts(t *testing.T) {
	var out bytes.Buffer
	if err := RenderKitBadges(&out, []Kit{KitSearch, KitSEO}); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if strings.Index(html, "search") > strings.Index(html, "seo") || !strings.Contains(html, `aria-label="Etkin yetenekler"`) {
		t.Fatalf("badges = %s", html)
	}
}
