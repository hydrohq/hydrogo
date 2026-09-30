package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNotFoundPathsEscapesRoutesAndShowsConversionForm(t *testing.T) {
	var out bytes.Buffer
	if err := NotFoundPaths([]NotFoundPathRow{{Path: `/<script>`, Hits: "7", LastSeen: "2026-09-29 12:00"}}, "csrf-token").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"&lt;script&gt;", "7", "/admin/redirects/convert", "csrf-token", "301 yönlendirmesi oluştur"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "<script>") {
		t.Fatal("untrusted path was rendered as markup")
	}
}
