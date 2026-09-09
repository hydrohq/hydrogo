package media

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type httpAssetRepo struct{ asset Asset }

func (r httpAssetRepo) Get(context.Context, database.Executor, string) (Asset, error) {
	return r.asset, nil
}
func (r httpAssetRepo) CreatePending(context.Context, database.Executor, Asset) error { return nil }
func (r httpAssetRepo) MarkReady(context.Context, database.Executor, string, string) error {
	return nil
}

func TestPrivateMediaIsNotServedWithoutSession(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "private.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	handler := HTTPHandler{Service: Service{Uploader: uploader, Assets: httpAssetRepo{asset: Asset{ID: "a1", StorageKey: "private.txt", MimeType: "text/plain", Status: "ready", Visibility: VisibilityPrivate}}}}
	response := httptest.NewRecorder()
	handler.serve(response, httptest.NewRequest(http.MethodGet, "/media/a1", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("private media status = %d, want 401", response.Code)
	}
}

func TestMultipartBodyLimitDoesNotOverflow(t *testing.T) {
	const maxInt64 = int64(^uint64(0) >> 1)
	if got := multipartBodyLimit(maxInt64 - 1); got != maxInt64 {
		t.Fatalf("near-max multipart limit = %d, want %d", got, maxInt64)
	}
	if got := multipartBodyLimit(1 << 20); got != 2<<20 {
		t.Fatalf("multipart limit = %d, want %d", got, 2<<20)
	}
}

func TestPublicMediaIsServedWhenReady(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "public.txt"), []byte("public"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	handler := HTTPHandler{Service: Service{Uploader: uploader, Assets: httpAssetRepo{asset: Asset{ID: "a2", StorageKey: "public.txt", MimeType: "text/plain", Status: "ready", Visibility: VisibilityPublic}}}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/media/a2", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "u1", Active: true}))
	handler.serve(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "public" {
		t.Fatalf("public media response = %d/%q", response.Code, response.Body.String())
	}
}

func TestMediaDoesNotServeSymlinkedStorageEntry(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret outside storage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "linked.txt")); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	handler := HTTPHandler{Service: Service{Uploader: uploader, Assets: httpAssetRepo{asset: Asset{ID: "a3", StorageKey: "linked.txt", MimeType: "text/plain", Status: "ready", Visibility: VisibilityPublic}}}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/media/a3", nil)
	handler.serve(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("symlinked media status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestMediaHTTPFailsClosedWhenUploaderIsMissing(t *testing.T) {
	handler := HTTPHandler{}
	request := httptest.NewRequest(http.MethodPost, "/api/media", bytes.NewReader(nil))
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
	response := httptest.NewRecorder()
	handler.upload(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing uploader upload status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}

	serveResponse := httptest.NewRecorder()
	handler.serve(serveResponse, httptest.NewRequest(http.MethodGet, "/media/id", nil))
	if serveResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing uploader serve status = %d, want %d", serveResponse.Code, http.StatusServiceUnavailable)
	}
}

func TestMediaHTTPFailsClosedWhenRepositoryIsMissing(t *testing.T) {
	uploader, err := NewUploader(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "image.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write(createTestPNG(2, 2))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/media", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
	response := httptest.NewRecorder()
	(HTTPHandler{Service: Service{Uploader: uploader}}).upload(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing repository upload status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
