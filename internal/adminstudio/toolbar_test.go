package adminstudio

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderQueryToolbarEscapesAndBounds(t *testing.T) {
	var out bytes.Buffer
	err := RenderQueryToolbar(&out, QueryToolbarData{Action: "/admin/items", Query: ListQuery{Search: `<script>`}, FilterOptions: map[string][]string{"status": {"draft", "published"}}, SortOptions: []string{"-created_at"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<script>") || !strings.Contains(out.String(), "&lt;script&gt;") || !strings.Contains(out.String(), `name="filter[status]"`) {
		t.Fatalf("unsafe/incomplete toolbar: %s", out.String())
	}
}

func TestRenderQueryToolbarRejectsExternalAction(t *testing.T) {
	if err := RenderQueryToolbar(&bytes.Buffer{}, QueryToolbarData{Action: "https://evil.test"}); err == nil {
		t.Fatal("external action accepted")
	}
}
