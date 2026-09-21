package media

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type httpAssetRepo struct{ asset Asset }

type httpSignedBlobStore struct{ url string }

func (s httpSignedBlobStore) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (s httpSignedBlobStore) Get(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (s httpSignedBlobStore) Delete(context.Context, string) error { return nil }
func (s httpSignedBlobStore) PresignedURL(context.Context, string, time.Duration) (string, error) {
	return s.url, nil
}

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

func TestPrivateMediaLiveHTTPRejectsAnonymous(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "private-live.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	handler := HTTPHandler{Service: Service{
		Uploader: uploader,
		Assets:   httpAssetRepo{asset: Asset{ID: "live-private", StorageKey: "private-live.txt", MimeType: "text/plain", Status: "ready", Visibility: VisibilityPrivate}},
	}}
	mux := http.NewServeMux()
	mux.Handle("GET /media/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Model the real authentication middleware boundary while keeping this
		// network test independent of a session store.
		if r.Header.Get("X-Test-Editor") == "1" {
			r = r.WithContext(auth.WithUser(r.Context(), &auth.User{ID: "live-editor", Active: true, Role: auth.RoleEditor}))
		}
		handler.serve(w, r)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/media/live-private")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous live media status = %d, want 401", response.StatusCode)
	}
	authorizedRequest, err := http.NewRequest(http.MethodGet, server.URL+"/media/live-private", nil)
	if err != nil {
		t.Fatal(err)
	}
	authorizedRequest.Header.Set("X-Test-Editor", "1")
	response, err = server.Client().Do(authorizedRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authorized live media status = %d, want 200", response.StatusCode)
	}
}

func TestPrivateMediaIsNotServedToViewer(t *testing.T) {
	handler := HTTPHandler{Service: Service{Uploader: &Uploader{}, Assets: httpAssetRepo{asset: Asset{Status: "ready", Visibility: VisibilityPrivate}}}}
	request := httptest.NewRequest(http.MethodGet, "/media/a1", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "viewer", Active: true, Role: auth.RoleViewer}))
	response := httptest.NewRecorder()
	handler.serve(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("private media viewer status = %d, want 401", response.Code)
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

func TestMediaSignedURLRequiresEditorAndReturnsBoundedURL(t *testing.T) {
	asset := Asset{ID: "a-url", StorageKey: "images/a.png", MimeType: "image/png", Status: "ready", Visibility: VisibilityPrivate}
	handler := HTTPHandler{Service: Service{BlobStore: httpSignedBlobStore{url: "https://cdn.example.test/images/a.png?sig=x"}, Assets: httpAssetRepo{asset: asset}}}
	request := httptest.NewRequest(http.MethodGet, "/api/media/a-url/url?expires=60", nil)
	response := httptest.NewRecorder()
	handler.url(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("anonymous signed URL status = %d", response.Code)
	}
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
	response = httptest.NewRecorder()
	handler.url(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "https://cdn.example.test") || !strings.Contains(response.Body.String(), `"expires_in":60`) {
		t.Fatalf("signed URL response = %d/%s", response.Code, response.Body.String())
	}
}

func TestMediaSignedURLRejectsInvalidExpiryOrMissingProvider(t *testing.T) {
	asset := Asset{ID: "a-url", StorageKey: "images/a.png", Status: "ready", Visibility: VisibilityPublic}
	user := &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}
	for _, raw := range []string{"0", "86401", "invalid"} {
		handler := HTTPHandler{Service: Service{BlobStore: httpSignedBlobStore{url: "https://cdn.example.test/a.png"}, Assets: httpAssetRepo{asset: asset}}}
		request := httptest.NewRequest(http.MethodGet, "/api/media/a-url/url?expires="+raw, nil).WithContext(auth.WithUser(context.Background(), user))
		response := httptest.NewRecorder()
		handler.url(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("expiry %q status = %d", raw, response.Code)
		}
	}
	handler := HTTPHandler{Service: Service{Assets: httpAssetRepo{asset: asset}}}
	request := httptest.NewRequest(http.MethodGet, "/api/media/a-url/url", nil).WithContext(auth.WithUser(context.Background(), user))
	response := httptest.NewRecorder()
	handler.url(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing signer status = %d", response.Code)
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

func TestMediaUploadRejectsMalformedMultipart(t *testing.T) {
	uploader, err := NewUploader(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/media", bytes.NewBufferString("--broken\r\nnot-a-complete-part"))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
	response := httptest.NewRecorder()
	(HTTPHandler{Service: Service{Uploader: uploader, Assets: httpAssetRepo{}}}).upload(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("malformed multipart status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestMediaUploadRejectsMultipartBodyBeyondBound(t *testing.T) {
	uploader, err := NewUploader(t.TempDir(), 8)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/media", bytes.NewBufferString(strings.Repeat("x", 2<<20)))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
	response := httptest.NewRecorder()
	(HTTPHandler{Service: Service{Uploader: uploader, Assets: httpAssetRepo{}}}).upload(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized multipart status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
