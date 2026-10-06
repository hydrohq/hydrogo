package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdminShellLoadsAccessibleStatusFilter(t *testing.T) {
	var output bytes.Buffer
	if err := AdminStartLocaleRole(&output, "Pages", "", "pages", "en", "admin"); err != nil {
		t.Fatal(err)
	}
	markup := output.String()
	for _, want := range []string{".status-pill--published", "data-status-filter", "data-status-row", "aria-live", "data-status-filter-clear", "data-copy-text", "clipboard.writeText"} {
		if !strings.Contains(markup, want) {
			t.Errorf("admin shell omitted status filtering support %q", want)
		}
	}
}
