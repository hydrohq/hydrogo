package transport

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestAllowHeadStylesHashesOnlyMachineOwnedStyles(t *testing.T) {
	head, err := renderString(context.Background(), seoHead(seoHeadView{AppearanceCSS: ":root{--brand:#155EEF}"}))
	if err != nil {
		t.Fatal(err)
	}
	_, styles := security.InlineCSPHashes([]byte(head))
	if len(styles) != 1 {
		t.Fatalf("head styles = %v in %s", styles, head)
	}

	strict := http.Header{}
	strict.Set("Content-Security-Policy", "default-src 'self'; style-src 'self'")
	allowHeadStyles(strict, head)
	if policy := strict.Get("Content-Security-Policy"); !strings.Contains(policy, "'sha256-"+styles[0]+"'") {
		t.Fatalf("strict policy missing the appearance hash: %s", policy)
	}

	relaxed := http.Header{}
	relaxed.Set("Content-Security-Policy", "style-src 'self' 'unsafe-inline'")
	allowHeadStyles(relaxed, head)
	if policy := relaxed.Get("Content-Security-Policy"); strings.Contains(policy, "sha256-") {
		t.Fatalf("a hash would disable 'unsafe-inline': %s", policy)
	}

	unset := http.Header{}
	allowHeadStyles(unset, head)
	if policy := unset.Get("Content-Security-Policy"); policy != "" {
		t.Fatalf("an unset policy must stay unset, got %s", policy)
	}
}
