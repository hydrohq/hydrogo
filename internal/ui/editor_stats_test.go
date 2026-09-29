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
	for _, want := range []string{`data-editor-stats`, `aria-live="polite"`, `data-editor-words`, `data-editor-characters`, `data-editor-reading-time`, `data-editor-keyword-density`, `data-seo-audit`, `data-seo-score`, `data-seo-title`, `data-seo-description`, `data-seo-heading`, `data-seo-alt`, `data-seo-readability`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stats panel missing %q: %s", want, out.String())
		}
	}
	for _, want := range []string{"titleLength>=30&&titleLength<=60", "descriptionLength>=120&&descriptionLength<=160", "hasH1&&!skipped", "missingAlt===0", "average<=25"} {
		if !strings.Contains(AdminEditorStatsJS, want) {
			t.Errorf("SEO audit behavior missing %q", want)
		}
	}
}
