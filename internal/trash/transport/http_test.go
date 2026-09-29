package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
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
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("move pages status=%d want 303", rec.Code)
		}
		if len(repo.moved) != 1 || repo.moved[0] != "pages:p-1" {
			t.Fatalf("page not moved: %v", repo.moved)
		}
	})

	t.Run("site admin can trash submissions", func(t *testing.T) {
		rec, repo := performTrashAction(t, admin, "submissions", "f-1", "delete")
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("admin move submissions status=%d want 303", rec.Code)
		}
		if len(repo.moved) != 1 || repo.moved[0] != "submissions:f-1" {
			t.Fatalf("submission not moved by admin: %v", repo.moved)
		}
	})
}

func performTrashAction(t *testing.T, user *auth.User, kind, id, action string) (*httptest.ResponseRecorder, *fakeRepository) {
	t.Helper()
	repo := &fakeRepository{}
	handler := Handler{Service: trash.Service{Repository: repo}}
	req := httptest.NewRequest(http.MethodPost, "/admin/trash/"+kind+"/"+id+"/"+action, nil)
	req = req.WithContext(auth.WithUser(context.Background(), user))
	req.SetPathValue("kind", kind)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	if action == "delete" {
		handler.move(rec, req)
	} else {
		handler.restore(rec, req)
	}
	return rec, repo
}
