package adminstudio

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderListShowsOnlyConfiguredColumnsAndEscapesRows(t *testing.T) {
	collection := Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}, {ID: "internal", Label: "Internal", Kind: FieldText}, {ID: "status", Label: "Status", Kind: FieldStatus, VisibleInList: true}}, Actions: []Action{{Label: "Catalog", Href: "/admin/commerce/products"}}}
	var out bytes.Buffer
	err := RenderList(&out, ListData{Collection: collection, Rows: []Row{{Values: map[string]string{"title": `<img src=x>`, "internal": "hidden", "status": "published"}, Actions: []Action{{Label: "Variants", Href: "/admin/commerce/products/p1/variants"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"<caption class=\"visually-hidden\">Products listesi</caption>", `<th scope="col">Title</th>`, `<th scope="col">Status</th>`, `data-label="Title"`, `data-label="İşlemler"`, "Variants", "&lt;img src=x&gt;"} {
		if !strings.Contains(html, want) {
			t.Errorf("list missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, "Internal") || strings.Contains(html, ">hidden<") || strings.Contains(html, "<img src=x>") {
		t.Fatalf("list leaked hidden or unsafe data: %s", html)
	}
}

func TestRenderListUsesTypedStatusPillsAndScheduleTime(t *testing.T) {
	collection := Collection{ID: "news", Label: "News", Fields: []Field{{ID: "status", Label: "Status", Kind: FieldStatus, VisibleInList: true}}, Actions: []Action{{Label: "Edit", Href: "/admin/news/n1"}}}
	scheduled := time.Date(2026, time.October, 15, 9, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	var out strings.Builder
	err := RenderList(&out, ListData{Collection: collection, Rows: []Row{{Values: map[string]string{"status": "scheduled"}, Status: StatusInfo{Value: "scheduled", ScheduledAt: &scheduled}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-status-row="scheduled"`, `Zamanlandı · 15 Oct 06:00 UTC`, `datetime="2026-10-15T06:00:00Z"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("status list omitted %q: %s", want, out.String())
		}
	}
}

func TestRenderListPreservesEditionSpecificStatusValues(t *testing.T) {
	collection := Collection{ID: "projects", Label: "Projects", Fields: []Field{{ID: "status", Label: "Status", Kind: FieldStatus, VisibleInList: true}}, Actions: []Action{{Label: "Edit", Href: "/admin/projects/p1"}}}
	var out strings.Builder
	if err := RenderList(&out, ListData{Collection: collection, Rows: []Row{{Values: map[string]string{"status": "active"}}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), " active</span>") || strings.Contains(out.String(), "Diğer") {
		t.Fatalf("Edition status was hidden behind a generic label: %s", out.String())
	}
}

func TestRenderListAppliesStatusFilterWithoutMutatingRows(t *testing.T) {
	collection := Collection{ID: "pages", Label: "Pages", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}, {ID: "status", Label: "Status", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Edit", Href: "/admin/pages/edit"}}}
	rows := []Row{{Values: map[string]string{"title": "Published page", "status": "published"}}, {Values: map[string]string{"title": "Draft page", "status": "draft"}}}
	var out strings.Builder
	if err := RenderList(&out, ListData{Collection: collection, Rows: rows, StatusFilter: " DRAFT "}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Published page") || !strings.Contains(out.String(), "Draft page") {
		t.Fatalf("status filter mismatch: %s", out.String())
	}
	if len(rows) != 2 {
		t.Fatalf("RenderList mutated input rows: %d", len(rows))
	}
}

func TestRenderListCopyActionEscapesURLAndUsesTypedButton(t *testing.T) {
	collection := Collection{ID: "pages", Label: "Pages", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Edit", Href: "/admin/pages/p1"}}}
	var out strings.Builder
	if err := RenderList(&out, ListData{Collection: collection, Rows: []Row{{Values: map[string]string{"title": "Home"}, Actions: []Action{{Label: "Copy live URL", CopyText: `https://site.example/path?q="x"`}}}}}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-copy-text="https://site.example/path?q=&#34;x&#34;"`, `aria-label="Copy live URL"`, `data-copy-status`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("copy action omitted %q: %s", want, out.String())
		}
	}
}

func TestRenderListProvidesAccessibleEmptyState(t *testing.T) {
	collection := Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/products/new"}}}
	var out bytes.Buffer
	if err := RenderList(&out, ListData{Collection: collection, EmptyMessage: "Henüz ürün eklenmedi."}); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`colspan="2"`, `class="empty-state"`, `role="status"`, "Henüz ürün eklenmedi."} {
		if !strings.Contains(html, want) {
			t.Errorf("empty state missing %q: %s", want, html)
		}
	}
}

func TestRenderListUsesSafeDefaultEmptyMessage(t *testing.T) {
	var out bytes.Buffer
	if err := RenderList(&out, ListData{Collection: Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/products/new"}}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Bu koleksiyonda henüz kayıt bulunmuyor.") {
		t.Fatalf("default empty message missing: %s", out.String())
	}
}

func TestRenderListDistinguishesEmptyCollectionFromNoMatchesInSelectedLanguage(t *testing.T) {
	collection := Collection{ID: "products", Label: "Products", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Edit", Href: "/admin/products/p1"}}}
	for _, tc := range []struct {
		name   string
		locale string
		query  ListQuery
		want   string
	}{
		{name: "Turkish empty", locale: "tr", want: "Bu koleksiyonda henüz kayıt bulunmuyor."},
		{name: "English empty", locale: "en", want: "There are no records in this collection yet."},
		{name: "Turkish search and filter", locale: "tr", query: ListQuery{Search: "missing", Filters: map[string]string{"status": "active"}}, want: "Aramanızı veya filtrelerinizi değiştirip tekrar deneyin. Eşleşen kayıt yok."},
		{name: "English search", locale: "en", query: ListQuery{Search: "missing"}, want: "Try changing your search or filters. No records match them."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := RenderList(&out, ListData{Collection: collection, UILocale: tc.locale, Query: tc.query}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), `role="status"`) || !strings.Contains(out.String(), tc.want) {
				t.Fatalf("list empty status missing localized guidance %q: %s", tc.want, out.String())
			}
		})
	}
}

func TestRenderListOptInBulkSelection(t *testing.T) {
	collection := Collection{ID: "articles", Label: "Articles", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/articles/new"}}, BulkActions: []Action{{Label: "Yayınla", Href: "/admin/articles/bulk/publish"}}}
	var out strings.Builder
	if err := RenderList(&out, ListData{Collection: collection, Rows: []Row{{ID: "a1", Values: map[string]string{"title": "Hello"}}}}); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`data-bulk-select-all`, `value="a1"`, `data-bulk-action="Yayınla"`} {
		if !strings.Contains(out.String(), marker) {
			t.Fatalf("bulk list missing %q: %s", marker, out.String())
		}
	}
}

func TestRenderListBulkSelectionRequiresRowID(t *testing.T) {
	collection := Collection{ID: "articles", Label: "Articles", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Yeni", Href: "/admin/articles/new"}}, BulkActions: []Action{{Label: "Yayınla", Href: "/admin/articles/bulk/publish"}}}
	if err := RenderList(&strings.Builder{}, ListData{Collection: collection, Rows: []Row{{Values: map[string]string{"title": "Hello"}}}}); err != ErrInvalidForm {
		t.Fatalf("missing row ID error = %v", err)
	}
}
