package ui

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func TestRenderAdminAuditEscapesDetailsAndShowsFilters(t *testing.T) {
	var output strings.Builder
	err := RenderAdminAudit(&output, "nonce", "en", designsystem.AdminAuditPageProps{
		Action: "pages.updated", Total: 1, Limit: 50,
		Entries: []designsystem.AdminAuditEntry{{At: "29 Sep 2026", Actor: "admin@example.test", Action: "pages.updated", Resource: "pages", Target: "Page", BeforeJSON: `{"title":"<old>"}`, AfterJSON: `{"title":"new"}`}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/admin/audit", "pages.updated", "admin@example.test", "&lt;old&gt;", "View details"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("rendered audit page missing %q", want)
		}
	}
	if strings.Contains(output.String(), `"<old>"`) {
		t.Fatal("unescaped audit details rendered")
	}
}
