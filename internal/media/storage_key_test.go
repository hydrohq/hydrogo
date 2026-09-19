package media

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type recordingBlobStore struct{ key string }

func (r *recordingBlobStore) Put(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	r.key = key
	return nil
}
func (*recordingBlobStore) Get(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (*recordingBlobStore) Delete(context.Context, string) error               { return nil }
func (*recordingBlobStore) PresignedURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

type recordingAssetRepository struct{ asset Asset }

func (r *recordingAssetRepository) Get(context.Context, database.Executor, string) (Asset, error) {
	return r.asset, nil
}
func (r *recordingAssetRepository) CreatePending(_ context.Context, _ database.Executor, asset Asset) error {
	r.asset = asset
	return nil
}
func (r *recordingAssetRepository) MarkReady(_ context.Context, _ database.Executor, _ string, visibility string) error {
	r.asset.Visibility, r.asset.Status = visibility, "ready"
	return nil
}

func TestStorageKeyForContextNamespacesTenantObjects(t *testing.T) {
	if got, err := StorageKeyForContext(context.Background(), "asset.png"); err != nil || got != "asset.png" {
		t.Fatalf("legacy key = %q err=%v", got, err)
	}
	scope, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := StorageKeyForContext(tenant.WithScope(context.Background(), scope), "asset.png"); err != nil || got != "tenants/tenant-a/asset.png" {
		t.Fatalf("scoped key = %q err=%v", got, err)
	}
}

func TestServiceUploadUsesTenantObjectNamespace(t *testing.T) {
	uploader, err := NewUploader(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), database.SQLite, "file:media-storage-key?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := &recordingAssetRepository{}
	blob := &recordingBlobStore{}
	scope, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithScope(context.Background(), scope)
	service := Service{Uploader: uploader, Assets: repository, BlobStore: blob, Transactions: database.NewTransactor(db)}
	if _, err := service.Upload(ctx, bytesReader(createTestPNG(2, 2)), "asset.png"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(repository.asset.StorageKey, "tenants/tenant-a/") || blob.key != repository.asset.StorageKey {
		t.Fatalf("asset key=%q blob key=%q", repository.asset.StorageKey, blob.key)
	}
}

func bytesReader(data []byte) io.Reader { return &reader{data: data} }

type reader struct {
	data []byte
	pos  int
}

func (r *reader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
