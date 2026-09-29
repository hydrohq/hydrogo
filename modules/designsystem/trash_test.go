package designsystem

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/trash"
)

func TestTrashPageEscapesLabelsAndCarriesCSRF(t *testing.T) {
	var output bytes.Buffer
	items := []trash.Entry{{Kind: trash.Pages, ID: "page<1>", Label: `<script>alert(1)</script>`, DeletedAt: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), Trashed: true}}
	if err := TrashPage(items, "csrf-safe").Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`&lt;script&gt;alert(1)&lt;/script&gt;`, `name="_csrf" value="csrf-safe"`, `action="/admin/trash/pages/page%3C1%3E/restore"`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("trash page missing %q", want)
		}
	}
	if strings.Contains(output.String(), `<script>alert(1)</script>`) {
		t.Fatal("untrusted content label was rendered as markup")
	}
}
