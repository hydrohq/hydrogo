package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type memoryBlobStore struct {
	objects map[string][]byte
}

func newMemoryBlobStore() *memoryBlobStore {
	return &memoryBlobStore{objects: make(map[string][]byte)}
}

func (m *memoryBlobStore) Put(_ context.Context, key string, r io.Reader, size int64, _ string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.objects[key] = data
	return nil
}

func (m *memoryBlobStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memoryBlobStore) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

type mockQuarantineSigner struct {
	*memoryBlobStore
}

func (m *mockQuarantineSigner) PresignedQuarantineUploadURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	return "https://s3.example.test/bucket/" + key + "?signed=true&expires=" + expiry.String(), nil
}

func TestUploadURLAndConfirmUpload(t *testing.T) {
	memStore := newMemoryBlobStore()
	signer := &mockQuarantineSigner{memoryBlobStore: memStore}
	repo := &mockDirectUploadAssetRepo{assets: make(map[string]Asset)}

	db, err := database.Open(context.Background(), database.SQLite, "file:upload-url-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	svc := Service{
		BlobStore:    signer,
		Assets:       repo,
		Transactions: database.NewTransactor(db),
		Uploader: &Uploader{
			MaxBytes: 10 << 20, // 10MB
		},
	}
	h := HTTPHandler{
		Service: svc,
		DB:      db,
	}

	mux := http.NewServeMux()
	for _, route := range h.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}

	editor := &auth.User{ID: "editor1", Role: auth.RoleEditor, Active: true}

	// 1. Unauthenticated request rejected
	req := httptest.NewRequest("POST", "/api/v1/media/upload-url", strings.NewReader(`{"filename":"test.png"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unauthenticated request, got %d", w.Code)
	}

	// 2. Disallowed extension rejected
	req = httptest.NewRequest("POST", "/api/v1/media/upload-url", strings.NewReader(`{"filename":"danger.exe"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUser(context.Background(), editor))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for disallowed extension, got %d", w.Code)
	}

	// 3. Oversized request rejected
	req = httptest.NewRequest("POST", "/api/v1/media/upload-url", strings.NewReader(`{"filename":"big.png","size_bytes":20000000}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUser(context.Background(), editor))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for oversized file, got %d", w.Code)
	}

	// 4. Valid upload URL request
	req = httptest.NewRequest("POST", "/api/v1/media/upload-url", strings.NewReader(`{"filename":"photo.png","mime_type":"image/png","size_bytes":1024}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUser(context.Background(), editor))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid upload-url request, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data struct {
			AssetID   string `json:"asset_id"`
			ObjectKey string `json:"object_key"`
			UploadURL string `json:"upload_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if res.Data.AssetID == "" || res.Data.ObjectKey == "" || res.Data.UploadURL == "" {
		t.Fatalf("expected asset_id, object_key, upload_url in response: %+v", res.Data)
	}

	// Verify pending asset was stored
	if asset, ok := repo.assets[res.Data.AssetID]; !ok || asset.Status != "pending" {
		t.Fatalf("expected pending asset stored in repository, got %+v", repo.assets)
	}

	// 5. Direct upload simulation: simulate client uploading directly to S3
	payload := []byte("hello png content")
	_ = memStore.Put(context.Background(), res.Data.ObjectKey, bytes.NewReader(payload), int64(len(payload)), "image/png")

	// 6. Confirm upload
	confirmBody := `{"asset_id":"` + res.Data.AssetID + `","object_key":"` + res.Data.ObjectKey + `","size_bytes":` + jsonNumber(len(payload)) + `,"visibility":"public"}`
	req = httptest.NewRequest("POST", "/api/v1/media/confirm", strings.NewReader(confirmBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUser(context.Background(), editor))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for confirm, got %d: %s", w.Code, w.Body.String())
	}

	if asset, ok := repo.assets[res.Data.AssetID]; !ok || asset.Status != "ready" || asset.Visibility != "public" {
		t.Fatalf("expected asset promoted to ready and public, got %+v", asset)
	}
}

func jsonNumber(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

type mockDirectUploadAssetRepo struct {
	assets map[string]Asset
}

func (m *mockDirectUploadAssetRepo) Get(_ context.Context, _ database.Executor, id string) (Asset, error) {
	asset, ok := m.assets[id]
	if !ok {
		return Asset{}, errors.New("not found")
	}
	return asset, nil
}

func (m *mockDirectUploadAssetRepo) CreatePending(_ context.Context, _ database.Executor, asset Asset) error {
	m.assets[asset.ID] = asset
	return nil
}

func (m *mockDirectUploadAssetRepo) MarkReady(_ context.Context, _ database.Executor, id, visibility string) error {
	asset, ok := m.assets[id]
	if !ok {
		return errors.New("not found")
	}
	asset.Status = "ready"
	asset.Visibility = visibility
	m.assets[id] = asset
	return nil
}

func (m *mockDirectUploadAssetRepo) MarkReadyVerified(_ context.Context, _ database.Executor, id, storageKey, visibility string) error {
	asset, ok := m.assets[id]
	if !ok {
		return errors.New("not found")
	}
	if asset.StorageKey != storageKey {
		return errors.New("storage key mismatch")
	}
	asset.Status = "ready"
	asset.Visibility = visibility
	m.assets[id] = asset
	return nil
}
