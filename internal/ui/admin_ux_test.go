package ui

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func TestBusinessOwnerDashboardKeepsDeveloperDetailsOut(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin", nil)
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "user", Role: auth.RoleEditor, Active: true}))
	response := httptest.NewRecorder()
	if err := RenderBusinessOwnerDashboard(response, req, map[string]bool{"pages": true, "news": true}); err != nil {
		t.Fatal(err)
	}
	got := response.Body.String()
	for _, text := range []string{`data-business-view`, `href="/admin/pages"`, `href="/admin/news"`, `data-studio-role="editor"`} {
		if !strings.Contains(got, text) {
			t.Errorf("business dashboard missing %q", text)
		}
	}
	for _, forbidden := range []string{"SQLite", "PostgreSQL", "UUID", "stack trace", `>Geliştirici &amp; API</span>`} {
		if strings.Contains(got, forbidden) {
			t.Errorf("business dashboard leaks %q", forbidden)
		}
	}
}
