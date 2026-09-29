package media

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"mime/multipart"
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
)

type httpAssetRepo struct{ asset Asset }

type adminSettingsRepoFake struct {
	settings AdminSettings
	found    bool
}

func (r *adminSettingsRepoFake) LoadAdminSettings(context.Context, database.Executor) (AdminSettings, bool, error) {
	return r.settings, r.found, nil
}
func (r *adminSettingsRepoFake) SaveAdminSettings(_ context.Context, _ database.Executor, value AdminSettings) error {
	r.settings, r.found = value, true
	return nil
}

type settingsAssetRepo struct{ httpAssetRepo }

func (settingsAssetRepo) CreateFolder(context.Context, database.Executor, Folder) error { return nil }
func (settingsAssetRepo) ListFolders(context.Context, database.Executor) ([]Folder, error) {
	return nil, nil
}

type noDerivativeAssetRepo struct{ httpAssetRepo }

func (noDerivativeAssetRepo) CreateProcessing(context.Context, database.Executor, Variant) error {
	return nil
}
func (noDerivativeAssetRepo) MarkResult(context.Context, database.Executor, string, string, int64, string) error {
	return nil
}
func (noDerivativeAssetRepo) GetReady(context.Context, database.Executor, string, string) (Variant, error) {
	return Variant{}, sql.ErrNoRows
}

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

func TestAdminMediaSettingsPersistAndApplyHotlinkProtection(t *testing.T) {
	repository := &adminSettingsRepoFake{}
	handler := HTTPHandler{Service: Service{Assets: settingsAssetRepo{}, Optimizer: &Optimizer{}}, SettingsRepository: repository, SettingsCache: NewAdminSettingsCache()}
	form := url.Values{
		"hotlink_protection": {"on"}, "allowed_referer_hosts": {"partner.example"},
		"watermark_enabled": {"on"}, "watermark_text": {"© HydroGo"}, "watermark_position": {"bottom-left"},
		"watermark_opacity": {"40"}, "watermark_margin": {"16"},
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/media/settings", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	handler.adminSaveSettings(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("save status = %d body=%s", response.Code, response.Body.String())
	}
	if !repository.found || !repository.settings.HotlinkProtection || repository.settings.WatermarkOpacity != .4 || repository.settings.WatermarkText != "© HydroGo" {
		t.Fatalf("saved media settings = %+v", repository.settings)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "public.png"), []byte("source-master"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	handler.Service = Service{Uploader: uploader, Assets: noDerivativeAssetRepo{httpAssetRepo{asset: Asset{ID: "public-asset", StorageKey: "public.png", OriginalFilename: "public.png", MimeType: "image/png", Status: "ready", Visibility: VisibilityPublic}}}}
	var serve http.Handler
	for _, route := range handler.Routes() {
		if route.Pattern == "GET /media/{id}" {
			serve = route.Handler
			break
		}
	}
	if serve == nil {
		t.Fatal("media route missing")
	}
	blocked := httptest.NewRecorder()
	blockedRequest := httptest.NewRequest(http.MethodGet, "/media/public-asset", nil)
	blockedRequest.Header.Set("Referer", "https://evil.example/article")
	serve.ServeHTTP(blocked, blockedRequest)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("hotlink response = %d, want 403", blocked.Code)
	}
	direct := httptest.NewRecorder()
	serve.ServeHTTP(direct, httptest.NewRequest(http.MethodGet, "/media/public-asset", nil))
	if direct.Code != http.StatusServiceUnavailable || strings.Contains(direct.Body.String(), "source-master") {
		t.Fatalf("watermarked public response = %d body=%q; expected processing response without exposing the master", direct.Code, direct.Body.String())
	}
	admin := httptest.NewRecorder()
	adminRequest := httptest.NewRequest(http.MethodGet, "/media/public-asset", nil)
	adminRequest = adminRequest.WithContext(auth.WithUser(adminRequest.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	serve.ServeHTTP(admin, adminRequest)
	if admin.Code != http.StatusOK || !strings.Contains(admin.Body.String(), "source-master") {
		t.Fatalf("authorized master response = %d body=%q", admin.Code, admin.Body.String())
	}
	if err := os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler.Service.Assets = httpAssetRepo{asset: Asset{ID: "public-video", StorageKey: "video.mp4", MimeType: "video/mp4", Status: "ready", Visibility: VisibilityPublic}}
	var mediaRoute http.Handler
	for _, route := range handler.Routes() {
		if route.Pattern == "GET /media/{id}" {
			mediaRoute = route.Handler
			break
		}
	}
	video := httptest.NewRecorder()
	videoRequest := httptest.NewRequest(http.MethodGet, "/media/public-video", nil)
	videoRequest.Header.Set("Referer", "https://evil.example/article")
	mediaRoute.ServeHTTP(video, videoRequest)
	if video.Code != http.StatusOK {
		t.Fatalf("video response under image hotlink protection = %d, want 200", video.Code)
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

func TestPrivateDocumentRequiresEditorAndDownloadsAsAttachment(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "resume.pdf"), []byte("%PDF-1.7"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	handler := HTTPHandler{Service: Service{Uploader: uploader, Assets: httpAssetRepo{asset: Asset{ID: "cv", StorageKey: "resume.pdf", OriginalFilename: "resume.pdf", MimeType: "application/pdf", Status: "ready", Visibility: VisibilityPrivate}}}}
	request := httptest.NewRequest(http.MethodGet, "/media/cv", nil)
	response := httptest.NewRecorder()
	handler.serve(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous document status=%d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/media/cv", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
	response = httptest.NewRecorder()
	handler.serve(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Disposition"), "attachment") || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("private document headers/status = %d %#v", response.Code, response.Header())
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

func TestReadMultipartUploadStreamsFileToTempAndValidatesHeader(t *testing.T) {
	directory := t.TempDir()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("folder_id", "folder-1"); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("file", `..\images\photo.png`)
	if err != nil {
		t.Fatal(err)
	}
	payload := createTestPNG(3, 2)
	if _, err := file.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/media", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	result, err := readMultipartUpload(request, directory, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(result.Path)
	if result.Filename != "photo.png" || result.Fields["folder_id"] != "folder-1" {
		t.Fatalf("multipart metadata = %+v", result)
	}
	contents, err := os.ReadFile(result.Path)
	if err != nil || !bytes.Equal(contents, payload) {
		t.Fatalf("streamed temp content mismatch: %v", err)
	}

	badDirectory := t.TempDir()
	var badBody bytes.Buffer
	badWriter := multipart.NewWriter(&badBody)
	badFile, err := badWriter.CreateFormFile("file", "payload.bin")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = badFile.Write([]byte("not an allowed payload"))
	_ = badWriter.Close()
	badRequest := httptest.NewRequest(http.MethodPost, "/admin/media", &badBody)
	badRequest.Header.Set("Content-Type", badWriter.FormDataContentType())
	if _, err := readMultipartUpload(badRequest, badDirectory, 1<<20); !errors.Is(err, ErrDisallowedMimeType) {
		t.Fatalf("invalid initial MIME = %v", err)
	}
	entries, err := os.ReadDir(badDirectory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed upload left temp files: entries=%v err=%v", entries, err)
	}
}

type visibilityAssetRepo struct{ httpAssetRepo }

func (visibilityAssetRepo) SetVisibility(context.Context, database.Executor, string, string) error {
	return nil
}

func TestMediaReadyRejectsOversizedFormBody(t *testing.T) {
	uploader, err := NewUploader(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), database.SQLite, "file:media-ready-bound?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := HTTPHandler{Service: Service{Uploader: uploader, Assets: visibilityAssetRepo{httpAssetRepo{asset: Asset{ID: "a1", Status: "pending"}}}, Transactions: database.NewTransactor(db)}}
	assertReadyStatus := func(body string, want int) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/api/media/a1/ready", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}))
		response := httptest.NewRecorder()
		handler.ready(response, request)
		if response.Code != want {
			t.Fatalf("ready status = %d, want %d", response.Code, want)
		}
	}
	assertReadyStatus("visibility=public", http.StatusOK)
	assertReadyStatus("visibility=public&pad="+strings.Repeat("x", 16384), http.StatusBadRequest)
}
