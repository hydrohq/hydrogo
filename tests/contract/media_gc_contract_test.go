package contract_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

type gcBlobStore struct {
	deleted  []string
	failKey  string
	failures int
}

func (*gcBlobStore) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (*gcBlobStore) Get(context.Context, string) (io.ReadCloser, error)          { return nil, nil }
func (b *gcBlobStore) Delete(_ context.Context, key string) error {
	if key == b.failKey && b.failures > 0 {
		b.failures--
		return fmt.Errorf("injected object deletion failure")
	}
	b.deleted = append(b.deleted, key)
	return nil
}

func TestSQLiteMediaGarbageCollectionContract(t *testing.T) {
	runMediaGarbageContract(t, database.SQLite, filepath.Join(t.TempDir(), "media-gc.sqlite"))
}

func TestPostgresMediaGarbageCollectionContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runMediaGarbageContract(t, database.Postgres, dsn)
}

func runMediaGarbageContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	scope, err := tenant.NewScope("gc-contract-tenant", "gc-contract")
	if err != nil {
		t.Fatal(err)
	}
	ctx = tenant.WithScope(ctx, scope)
	marker := fmt.Sprintf("gc-contract-%d", time.Now().UTC().UnixNano())
	store := mediastore.Store{Dialect: dialect, RequireTenantScope: true}
	asset := media.Asset{ID: marker, StorageKey: "sha256/" + marker + ".png", OriginalFilename: marker + ".png", MimeType: "image/png", SizeBytes: 42}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return store.CreatePending(ctx, tx, asset) }); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkReady(ctx, db, asset.ID, media.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	variantKey := "sha256/" + marker + "-thumb.webp"
	variantInsert := `INSERT INTO media_variants (id,media_id,kind,storage_key,mime_type,size_bytes,status,created_at,updated_at,tenant_id) VALUES (?,?,?,?,?,?,'ready',?,?,?)`
	if dialect == database.Postgres {
		variantInsert = `INSERT INTO media_variants (id,media_id,kind,storage_key,mime_type,size_bytes,status,created_at,updated_at,tenant_id) VALUES ($1,$2,$3,$4,$5,$6,'ready',$7,$8,$9)`
	}
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, variantInsert, marker+"-thumb", marker, "thumbnail", variantKey, "image/webp", 16, now, now, scope.TenantID); err != nil {
		t.Fatal(err)
	}
	query := `UPDATE media_assets SET created_at = ? WHERE id = ?`
	if dialect == database.Postgres {
		query = `UPDATE media_assets SET created_at = $1 WHERE id = $2`
	}
	if _, err := db.ExecContext(ctx, query, time.Now().UTC().Add(-10*24*time.Hour).Format(time.RFC3339Nano), asset.ID); err != nil {
		t.Fatal(err)
	}
	storeRepo := mediastore.Store{Dialect: dialect, RequireTenantScope: true}
	blobs := &gcBlobStore{failKey: variantKey, failures: 1}
	collector := media.GarbageCollector{DB: db, Repository: storeRepo, References: mediastore.MediaReferences{Dialect: dialect, RequireTenantScope: true, Pages: true}, BlobStore: blobs, Retention: 7 * 24 * time.Hour}
	plan, err := collector.Plan(ctx)
	if err != nil || len(plan) != 1 || plan[0].ID != asset.ID || plan[0].SizeBytes != 58 {
		t.Fatalf("initial plan=%+v err=%v", plan, err)
	}
	pageID := "gc-page-" + marker
	pageInsert := `INSERT INTO pages (id,title,locale,slug,body,tenant_id) VALUES (?,?,?,?,?,?)`
	if dialect == database.Postgres {
		pageInsert = `INSERT INTO pages (id,title,locale,slug,body,tenant_id) VALUES ($1,$2,$3,$4,$5,$6)`
	}
	if _, err := db.ExecContext(ctx, pageInsert, pageID, "GC reference", "en", pageID, `{"media_asset_id":"`+asset.ID+`"}`, scope.TenantID); err != nil {
		t.Fatal(err)
	}
	plan, err = collector.Plan(ctx)
	if err != nil || len(plan) != 0 {
		t.Fatalf("referenced plan=%+v err=%v", plan, err)
	}
	pageDelete := `DELETE FROM pages WHERE id = ?`
	if dialect == database.Postgres {
		pageDelete = `DELETE FROM pages WHERE id = $1`
	}
	if _, err := db.ExecContext(ctx, pageDelete, pageID); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	for _, route := range (media.HTTPHandler{Garbage: &collector}).AdminRoutes() {
		if route.Pattern == "GET /admin/media/cleanup" || route.Pattern == "POST /admin/media/cleanup/delete" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	handler := security.CSRFMiddleware(false, nil)(mux)
	admin := &auth.User{ID: "gc-admin", Role: auth.RoleAdmin, Active: true}
	getCleanup := httptest.NewRequest(http.MethodGet, "/admin/media/cleanup", nil)
	getCleanup = getCleanup.WithContext(auth.WithUser(getCleanup.Context(), admin))
	getCleanup = getCleanup.WithContext(tenant.WithScope(getCleanup.Context(), scope))
	cleanupPage := httptest.NewRecorder()
	handler.ServeHTTP(cleanupPage, getCleanup)
	if cleanupPage.Code != http.StatusOK || !strings.Contains(cleanupPage.Body.String(), marker+".png") {
		t.Fatalf("cleanup page status=%d body=%q", cleanupPage.Code, cleanupPage.Body.String())
	}
	postCleanup := func(token string) *httptest.ResponseRecorder {
		form := url.Values{"_csrf": {token}, "asset_id": {asset.ID}, "confirm": {"yes"}}
		request := httptest.NewRequest(http.MethodPost, "/admin/media/cleanup/delete", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request = request.WithContext(auth.WithUser(request.Context(), admin))
		request = request.WithContext(tenant.WithScope(request.Context(), scope))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	invalid := postCleanup("wrong-token")
	if invalid.Code != http.StatusForbidden || len(blobs.deleted) != 0 {
		t.Fatalf("invalid CSRF status=%d blobs=%v", invalid.Code, blobs.deleted)
	}
	if _, err := store.Get(ctx, db, asset.ID); err != nil {
		t.Fatalf("invalid CSRF removed media metadata: %v", err)
	}
	failed := postCleanup("valid-token")
	if failed.Code != http.StatusServiceUnavailable || len(blobs.deleted) != 1 || blobs.deleted[0] != asset.StorageKey {
		t.Fatalf("first cleanup status=%d blobs=%v body=%q", failed.Code, blobs.deleted, failed.Body.String())
	}
	claimed, err := store.Get(ctx, db, asset.ID)
	if err != nil || claimed.Status != "deleting" {
		t.Fatalf("failed cleanup asset=%+v err=%v, want retryable deleting state", claimed, err)
	}
	retried := postCleanup("valid-token")
	if retried.Code != http.StatusSeeOther || len(blobs.deleted) != 3 || blobs.deleted[1] != asset.StorageKey || blobs.deleted[2] != variantKey {
		t.Fatalf("retry cleanup status=%d blobs=%v body=%q", retried.Code, blobs.deleted, retried.Body.String())
	}
	if _, err := store.Get(ctx, db, asset.ID); err == nil {
		t.Fatal("retried cleanup left a media metadata record")
	}
	var remainingVariants int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM media_variants WHERE media_id=`+map[bool]string{true: "$1", false: "?"}[dialect == database.Postgres], asset.ID).Scan(&remainingVariants); err != nil || remainingVariants != 0 {
		t.Fatalf("remaining variants=%d err=%v", remainingVariants, err)
	}
	deleted, err := collector.Delete(ctx, []string{asset.ID})
	if err != nil || deleted != 0 {
		t.Fatalf("repeat cleanup deleted=%d err=%v", deleted, err)
	}
	if len(blobs.deleted) != 3 {
		t.Fatalf("duplicate cleanup changed blob deletions: %v", blobs.deleted)
	}
}
