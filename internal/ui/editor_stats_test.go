package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestEditorStatsPanelUsesAccessibleTypedMarkup(t *testing.T) {
	var out bytes.Buffer
	if err := EditorStats().Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-editor-stats`, `aria-live="polite"`, `data-editor-words`, `data-editor-characters`, `data-editor-reading-time`, `data-editor-keyword-density`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stats panel missing %q: %s", want, out.String())
		}
	}
}
