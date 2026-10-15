package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
)

type fakeRepository struct {
	moved    []string
	restored []string
}

func (f *fakeRepository) List(context.Context, int) ([]trash.Entry, error) { return nil, nil }

func (f *fakeRepository) Move(_ context.Context, kind trash.Kind, id string, _ time.Time) error {
	f.moved = append(f.moved, string(kind)+":"+id)
	return nil
}

func (f *fakeRepository) Restore(_ context.Context, kind trash.Kind, id string, _ time.Time) error {
	f.restored = append(f.restored, string(kind)+":"+id)
	return nil
}

func TestTrashSubmissionsRequireSiteManagePermission(t *testing.T) {
	editor := &auth.User{ID: "editor-1", Role: auth.RoleEditor, Active: true}
	admin := &auth.User{ID: "admin-1", Role: auth.RoleAdmin, Active: true}

	t.Run("content editor cannot trash submissions", func(t *testing.T) {
		rec, repo := performTrashAction(t, editor, "submissions", "f-1", "delete")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("move submissions status=%d want 403 (role=%s)", rec.Code, editor.Role)
		}
		if len(repo.moved) != 0 {
			t.Fatalf("submission moved despite denial: %v", repo.moved)
		}
	})

	t.Run("content editor cannot restore submissions", func(t *testing.T) {
		rec, repo := performTrashAction(t, editor, "submissions", "f-1", "restore")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("restore submissions status=%d want 403", rec.Code)
		}
		if len(repo.restored) != 0 {
			t.Fatalf("submission restored despite denial: %v", repo.restored)
		}
	})

	t.Run("content editor can trash pages", func(t *testing.T) {
		rec, repo := performTrashAction(t, editor, "pages", "p-1", "delete")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("move pages status=%d want 204", rec.Code)
		}
		if len(repo.moved) != 1 || repo.moved[0] != "pages:p-1" {
			t.Fatalf("page not moved: %v", repo.moved)
		}
	})

	t.Run("site admin can trash submissions", func(t *testing.T) {
		rec, repo := performTrashAction(t, admin, "submissions", "f-1", "delete")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("admin move submissions status=%d want 204", rec.Code)
		}
		if len(repo.moved) != 1 || repo.moved[0] != "submissions:f-1" {
			t.Fatalf("submission not moved by admin: %v", repo.moved)
		}
	})

	t.Run("explicit content deny blocks editor and admin", func(t *testing.T) {
		for _, user := range []*auth.User{
			{ID: "editor-denied", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}},
			{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}},
		} {
			rec, repo := performTrashAction(t, user, "pages", "p-1", "delete")
			if rec.Code != http.StatusForbidden || len(repo.moved) != 0 {
				t.Fatalf("role=%s status=%d moved=%v", user.Role, rec.Code, repo.moved)
			}
		}
	})

	t.Run("explicit site grant permits submission management", func(t *testing.T) {
		user := &auth.User{ID: "editor-site-manager", Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: true}}
		rec, repo := performTrashAction(t, user, "submissions", "f-2", "restore")
		if rec.Code != http.StatusNoContent || len(repo.restored) != 1 || repo.restored[0] != "submissions:f-2" {
			t.Fatalf("status=%d restored=%v", rec.Code, repo.restored)
		}
	})

	t.Run("explicit site deny blocks admin submission management", func(t *testing.T) {
		user := &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}
		rec, repo := performTrashAction(t, user, "submissions", "f-2", "restore")
		if rec.Code != http.StatusForbidden || len(repo.restored) != 0 {
			t.Fatalf("status=%d restored=%v", rec.Code, repo.restored)
		}
	})
}

func TestTrashMutationRoutesEnforceCSRFBeforeService(t *testing.T) {
	for _, action := range []string{"delete", "restore"} {
		t.Run(action, func(t *testing.T) {
			for _, kind := range []string{"pages", "submissions"} {
				for _, csrfCase := range []struct {
					name  string
					token string
				}{
					{name: "missing"},
					{name: "mismatched", token: "wrong-token"},
				} {
					t.Run(kind+"/"+csrfCase.name, func(t *testing.T) {
						rec, repo := performTrashActionWithCSRF(t, &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}, kind, "item-1", action, csrfCase.token)
						if rec.Code != http.StatusForbidden || len(repo.moved)+len(repo.restored) != 0 {
							t.Fatalf("kind=%s csrf=%s status=%d moved=%v restored=%v", kind, csrfCase.name, rec.Code, repo.moved, repo.restored)
						}
					})
				}
			}
		})
	}
}

func performTrashAction(t *testing.T, user *auth.User, kind, id, action string) (*httptest.ResponseRecorder, *fakeRepository) {
	return performTrashActionWithCSRF(t, user, kind, id, action, "valid-token")
}

func performTrashActionWithCSRF(t *testing.T, user *auth.User, kind, id, action, token string) (*httptest.ResponseRecorder, *fakeRepository) {
	t.Helper()
	repo := &fakeRepository{}
	handler := Handler{Service: trash.Service{Repository: repo}}
	mux, err := router.New(handler.Routes(), func(r *http.Request) bool {
		actor := auth.UserFromContext(r.Context())
		return actor != nil && actor.Active
	})
	if err != nil {
		t.Fatal(err)
	}
	protected := security.CSRFMiddleware(false, nil)(mux)
	method, path := http.MethodDelete, AdminBasePath+"/"+kind+"/"+id
	if action == "restore" {
		method, path = http.MethodPost, path+"/restore"
	}
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
	if token != "" {
		req.Header.Set(security.CSRFHeaderName, token)
	}
	req = req.WithContext(auth.WithUser(context.Background(), user))
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	return rec, repo
}
