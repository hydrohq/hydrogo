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
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type newsAdminListRepository struct {
	news.Repository
	items []news.Item
	limit int
}

func (r *newsAdminListRepository) ListAdmin(_ context.Context, _ database.Executor, limit int) ([]news.Item, error) {
	r.limit = limit
	return r.items, nil
}

func TestNewsAdminListShowsReadAndWriteRoleActions(t *testing.T) {
	repository := &newsAdminListRepository{items: []news.Item{{
		ID: "news-1", Locale: "tr", Slug: "duyuru", Title: "Yeni duyuru", Status: news.StatusDraft,
	}}}
	handler := Handler{
		DB:      new(sql.DB),
		Service: news.Service{Repository: repository},
		AdminCollection: adminstudio.Collection{
			ID: "news", Label: "Haberler", TitleField: "title",
			Fields: []adminstudio.Field{
				{ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, VisibleInList: true},
				{ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, VisibleInList: true},
			},
			Actions: []adminstudio.Action{{Label: "Haberler", Href: "/admin/news"}},
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
	listTable := func(body string) string {
		t.Helper()
		start := strings.Index(body, `<table class="studio-table" data-admin-list-state>`)
		if start < 0 {
			t.Fatal("News Admin list table was not rendered")
		}
		end := strings.Index(body[start:], `</table>`)
		if end < 0 {
			t.Fatal("News Admin list table did not close")
		}
		return body[start : start+end+len(`</table>`)]
	}

	for _, tc := range []struct {
		name       string
		user       *auth.User
		wantAction string
		wantEdit   bool
	}{
		{name: "editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, wantAction: "Düzenle", wantEdit: true},
		{name: "viewer", user: &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}, wantAction: "Görüntüle"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := requestList(tc.user, "/admin/news")
			if response.Code != http.StatusOK {
				t.Fatalf("News list status=%d", response.Code)
			}
			table := listTable(response.Body.String())
			if !strings.Contains(table, "Yeni duyuru") || !strings.Contains(table, tc.wantAction) {
				t.Fatalf("News row or action missing: %s", table)
			}
			if gotEdit := strings.Contains(table, "Düzenle"); gotEdit != tc.wantEdit {
				t.Fatalf("edit action visible=%t, want %t", gotEdit, tc.wantEdit)
			}
		})
	}
	if repository.limit != 100 {
		t.Fatalf("News list limit=%d, want 100", repository.limit)
	}

	empty := requestList(&auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, "/admin/news?status=published")
	if empty.Code != http.StatusOK || !strings.Contains(empty.Body.String(), `role="status"`) || !strings.Contains(listTable(empty.Body.String()), "empty-state") {
		t.Fatalf("filtered empty News list is not accessible: status=%d", empty.Code)
	}
}
