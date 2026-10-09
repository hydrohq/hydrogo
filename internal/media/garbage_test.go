package media

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type garbageRepoStub struct {
	items    []GarbageAsset
	keys     []string
	claimed  map[string]bool
	deleted  []string
	restored []string
}

func (r *garbageRepoStub) ListGarbageCandidates(_ context.Context, _ database.Executor, before time.Time, _ int) ([]GarbageAsset, error) {
	var out []GarbageAsset
	for _, item := range r.items {
		if item.Status == "deleting" || item.CreatedAt.Before(before) {
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *garbageRepoStub) ClaimGarbageAsset(_ context.Context, _ database.Executor, id string, _ time.Time) (bool, error) {
	if r.claimed == nil {
		r.claimed = map[string]bool{}
	}
	r.claimed[id] = true
	return true, nil
}
func (r *garbageRepoStub) GarbageObjectKeys(context.Context, database.Executor, string) ([]string, error) {
	return r.keys, nil
}
func (r *garbageRepoStub) RestoreGarbageAsset(_ context.Context, _ database.Executor, id string) error {
	r.restored = append(r.restored, id)
	return nil
}
func (r *garbageRepoStub) DeleteGarbageAsset(_ context.Context, _ database.Executor, id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

type referencesStub struct {
	sets  []map[string]struct{}
	calls int
}

func (r *referencesStub) ReferencedMediaIDs(context.Context, database.Executor) (map[string]struct{}, error) {
	index := r.calls
	r.calls++
	if index >= len(r.sets) {
		index = len(r.sets) - 1
	}
	return r.sets[index], nil
}

type garbageBlobStub struct{ deleted []string }

func (*garbageBlobStub) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (*garbageBlobStub) Get(context.Context, string) (io.ReadCloser, error)          { return nil, nil }
func (b *garbageBlobStub) Delete(_ context.Context, key string) error {
	b.deleted = append(b.deleted, key)
	return nil
}

func TestGarbagePlanRequiresSevenDaysAndReferenceAbsence(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "gc.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	refs := &referencesStub{sets: []map[string]struct{}{{"used": {}}}}
	repo := &garbageRepoStub{items: []GarbageAsset{{ID: "unused", CreatedAt: now.Add(-10 * 24 * time.Hour), Status: "ready"}, {ID: "used", CreatedAt: now.Add(-10 * 24 * time.Hour), Status: "ready"}, {ID: "young", CreatedAt: now.Add(-time.Hour), Status: "ready"}}}
	collector := GarbageCollector{DB: db, Repository: repo, References: refs, Retention: 7 * 24 * time.Hour, Now: func() time.Time { return now }}
	items, err := collector.Plan(context.Background())
	if err != nil || len(items) != 1 || items[0].ID != "unused" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestGarbageDeleteRescansReferencesAfterClaimAndDeletesDerivatives(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "gc-delete.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	refs := &referencesStub{sets: []map[string]struct{}{{}, {}, {"orphan": {}}}}
	repo := &garbageRepoStub{items: []GarbageAsset{{ID: "orphan", CreatedAt: now.Add(-8 * 24 * time.Hour), Status: "ready"}}, keys: []string{"master.png", "thumb.png"}}
	blobs := &garbageBlobStub{}
	collector := GarbageCollector{DB: db, Repository: repo, References: refs, BlobStore: blobs, Retention: 7 * 24 * time.Hour}
	deleted, err := collector.Delete(context.Background(), []string{"orphan"})
	if err != nil || deleted != 0 || len(repo.restored) != 1 || len(repo.deleted) != 0 || len(blobs.deleted) != 0 {
		t.Fatalf("deleted=%d restored=%v metadata=%v blobs=%v err=%v", deleted, repo.restored, repo.deleted, blobs.deleted, err)
	}
	refs = &referencesStub{sets: []map[string]struct{}{{}, {}, {}}}
	collector.References = refs
	deleted, err = collector.Delete(context.Background(), []string{"orphan"})
	if err != nil || deleted != 1 || len(blobs.deleted) != 2 || len(repo.deleted) != 1 {
		t.Fatalf("deleted=%d metadata=%v blobs=%v err=%v", deleted, repo.deleted, blobs.deleted, err)
	}
}

func TestAdminMediaCleanupRequiresAdminAndConfirmation(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "gc-admin.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	repo := &garbageRepoStub{items: []GarbageAsset{{ID: "orphan", Filename: "old.png", SizeBytes: 12, CreatedAt: now.Add(-9 * 24 * time.Hour), Status: "ready"}}, keys: []string{"old.png"}}
	refs := &referencesStub{sets: []map[string]struct{}{{}, {}, {}}}
	collector := &GarbageCollector{DB: db, Repository: repo, References: refs, BlobStore: &garbageBlobStub{}, Retention: 7 * 24 * time.Hour}
	handler := HTTPHandler{Garbage: collector}
	adminCtx := auth.WithUser(context.Background(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true})
	page := httptest.NewRecorder()
	handler.adminCleanup(page, httptest.NewRequest(http.MethodGet, "/admin/media/cleanup", nil).WithContext(adminCtx))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "old.png") || !strings.Contains(page.Body.String(), "confirm") || !strings.Contains(page.Body.String(), "Kullanılmayan medya dosyalarını temizle") {
		t.Fatalf("GET status=%d body=%s", page.Code, page.Body.String())
	}
	form := url.Values{"asset_id": {"orphan"}, "confirm": {"yes"}}
	deniedAdminCtx := auth.WithUser(context.Background(), &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: false}})
	deniedRequest := httptest.NewRequest(http.MethodPost, "/admin/media/cleanup/delete", strings.NewReader(form.Encode())).WithContext(deniedAdminCtx)
	deniedRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deniedResponse := httptest.NewRecorder()
	handler.adminDeleteUnused(deniedResponse, deniedRequest)
	if deniedResponse.Code != http.StatusForbidden || len(repo.deleted) != 0 {
		t.Fatalf("media-denied Admin cleanup status=%d deleted=%v", deniedResponse.Code, repo.deleted)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/media/cleanup/delete", strings.NewReader(form.Encode())).WithContext(adminCtx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.adminDeleteUnused(response, req)
	if response.Code != http.StatusSeeOther || len(repo.deleted) != 1 {
		t.Fatalf("POST status=%d deleted=%v", response.Code, repo.deleted)
	}
	editorCtx := auth.WithUser(context.Background(), &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true})
	denied := httptest.NewRecorder()
	handler.adminCleanup(denied, httptest.NewRequest(http.MethodGet, "/admin/media/cleanup", nil).WithContext(editorCtx))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("editor status=%d", denied.Code)
	}
}

func TestAdminMediaCleanupRouteRequiresCSRFBeforeDeleting(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "gc-csrf.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now().UTC()
	repo := &garbageRepoStub{items: []GarbageAsset{{ID: "orphan", Filename: "old.png", SizeBytes: 12, CreatedAt: now.Add(-9 * 24 * time.Hour), Status: "ready"}}, keys: []string{"old.png"}}
	collector := &GarbageCollector{DB: db, Repository: repo, References: &referencesStub{sets: []map[string]struct{}{{}, {}, {}}}, BlobStore: &garbageBlobStub{}, Retention: 7 * 24 * time.Hour}
	mux := http.NewServeMux()
	for _, route := range (HTTPHandler{Garbage: collector}).AdminRoutes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	handler := security.CSRFMiddleware(false, nil)(mux)
	user := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	post := func(token string) *httptest.ResponseRecorder {
		form := url.Values{"_csrf": {token}, "asset_id": {"orphan"}, "confirm": {"yes"}}
		request := httptest.NewRequest(http.MethodPost, "/admin/media/cleanup/delete", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	invalid := post("wrong-token")
	if invalid.Code != http.StatusForbidden || len(repo.deleted) != 0 {
		t.Fatalf("invalid CSRF status=%d deleted=%v body=%q", invalid.Code, repo.deleted, invalid.Body.String())
	}
	valid := post("valid-token")
	if valid.Code != http.StatusSeeOther || len(repo.deleted) != 1 || repo.deleted[0] != "orphan" {
		t.Fatalf("valid CSRF status=%d deleted=%v body=%q", valid.Code, repo.deleted, valid.Body.String())
	}
}
