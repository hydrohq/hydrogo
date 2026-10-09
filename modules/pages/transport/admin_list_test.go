package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

type pageAdminListRepository struct {
	items []pages.Page
	limit int
}

func (r *pageAdminListRepository) GetContent(context.Context, database.Executor, string) (pages.Page, error) {
	return pages.Page{}, nil
}

func (r *pageAdminListRepository) GetPublishedByRoute(context.Context, database.Executor, string, string) (pages.Page, error) {
	return pages.Page{}, nil
}

func (r *pageAdminListRepository) ListPublished(context.Context, database.Executor, string, int) ([]pages.Page, error) {
	return nil, nil
}

func (r *pageAdminListRepository) CreateDraftContent(context.Context, database.Executor, pages.Page) error {
	return nil
}

func (r *pageAdminListRepository) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	return nil
}

func (r *pageAdminListRepository) Publish(context.Context, database.Executor, string, int) error {
	return nil
}

func (r *pageAdminListRepository) ListAdmin(_ context.Context, _ database.Executor, limit int) ([]pages.Page, error) {
	r.limit = limit
	return r.items, nil
}

func TestPageAdminListShowsRoleAppropriateActionsAndAccessibleEmptyState(t *testing.T) {
	repository := &pageAdminListRepository{items: []pages.Page{{
		ID: "page-1", Locale: "tr", Slug: "gizlilik", Title: "Gizlilik", Status: pages.StatusDraft,
	}}}
	handler := Handler{
		DB:      new(sql.DB),
		Content: service.ContentService{Pages: repository},
		AdminCollection: adminstudio.Collection{
			ID: "pages", Label: "Sayfalar", TitleField: "title",
			Fields: []adminstudio.Field{
				{ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, VisibleInList: true},
				{ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, VisibleInList: true},
			},
			Actions: []adminstudio.Action{{Label: "Yeni sayfa", Href: "/admin/pages/new"}},
		},
	}

	requestList := func(user *auth.User, target string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, target, nil)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		handler.adminList(response, request)
		return response
	}

	editorResponse := requestList(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "/admin/pages")
	if editorResponse.Code != http.StatusOK {
		t.Fatalf("Editor list status=%d", editorResponse.Code)
	}
	editorBody := editorResponse.Body.String()
	editorTable := pageAdminListTable(t, editorBody)
	for _, want := range []string{"Gizlilik", "Düzenle", `href="/admin/pages/page-1"`} {
		if !strings.Contains(editorTable, want) {
			t.Errorf("Editor list table missing %q", want)
		}
	}
	if !strings.Contains(editorBody, `href="/admin/pages/import"`) {
		t.Error("Editor list did not expose the permitted import action")
	}
	if repository.limit != 100 {
		t.Fatalf("Admin list limit=%d, want bounded limit 100", repository.limit)
	}

	viewerResponse := requestList(&auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}, "/admin/pages")
	if viewerResponse.Code != http.StatusOK {
		t.Fatalf("Viewer list status=%d", viewerResponse.Code)
	}
	viewerBody := viewerResponse.Body.String()
	viewerTable := pageAdminListTable(t, viewerBody)
	if !strings.Contains(viewerTable, "Görüntüle") || strings.Contains(viewerTable, "Düzenle") {
		t.Fatalf("Viewer list table exposed wrong action: %s", viewerTable)
	}
	if strings.Contains(viewerBody, `href="/admin/pages/import"`) {
		t.Fatal("Viewer list exposed the import action")
	}

	emptyResponse := requestList(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "/admin/pages?status=published")
	if emptyResponse.Code != http.StatusOK || !strings.Contains(emptyResponse.Body.String(), `role="status"`) || !strings.Contains(pageAdminListTable(t, emptyResponse.Body.String()), "empty-state") {
		t.Fatalf("filtered empty list lacks accessible status state: status=%d", emptyResponse.Code)
	}
}

func pageAdminListTable(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<table class="studio-table" data-admin-list-state>`)
	if start < 0 {
		t.Fatal("Admin page list table was not rendered")
	}
	end := strings.Index(body[start:], `</table>`)
	if end < 0 {
		t.Fatal("Admin page list table did not close")
	}
	return body[start : start+end+len(`</table>`)]
}
