package transport

import (
	"strings"
	"testing"
)

func TestLocalizePublicLinksRewritesStaticNavigation(t *testing.T) {
	markup := `<a href="/public/pages/tr/showcase/html">Showcase</a><a href="/public/news/tr/html">News</a>`
	localized := localizePublicLinks(markup, "en-US")
	for _, want := range []string{"/public/pages/en-US/showcase/html", "/public/news/en-US/html"} {
		if !strings.Contains(localized, want) {
			t.Fatalf("localized markup missing %q: %s", want, localized)
		}
	}
	if strings.Contains(localized, "/tr/") {
		t.Fatalf("localized markup retained Turkish route: %s", localized)
	}
}

func TestLocalizePublicLinksFallsBackToTurkish(t *testing.T) {
	if got := localizePublicLinks(`/public/news/tr/html`, ""); got != "/public/news/tr/html" {
		t.Fatalf("empty locale fallback = %q", got)
	}
}
