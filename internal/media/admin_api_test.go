package media

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type listingAssetRepo struct {
	httpAssetRepo
	assets   []Asset
	metadata map[string]AssetMetadata
	visible  map[string]string
}

func (r *listingAssetRepo) List(context.Context, database.Executor, int) ([]Asset, error) {
	return append([]Asset(nil), r.assets...), nil
}

func (r *listingAssetRepo) Get(_ context.Context, _ database.Executor, id string) (Asset, error) {
	for _, asset := range r.assets {
		if asset.ID == id {
			if metadata, ok := r.metadata[id]; ok {
				asset.FolderID, asset.Tags, asset.FocalX, asset.FocalY = metadata.FolderID, metadata.Tags, metadata.Focal.X, metadata.Focal.Y
			}
			if visibility, ok := r.visible[id]; ok {
				asset.Visibility = visibility
			}
			return asset, nil
		}
	}
	return Asset{}, ErrAssetRepositoryRequired
}

func (r *listingAssetRepo) SetMetadata(_ context.Context, _ database.Executor, id string, metadata AssetMetadata) error {
	if r.metadata == nil {
		r.metadata = map[string]AssetMetadata{}
	}
	r.metadata[id] = metadata
	return nil
}

func (r *listingAssetRepo) SetVisibility(_ context.Context, _ database.Executor, id, visibility string) error {
	if r.visible == nil {
		r.visible = map[string]string{}
	}
	r.visible[id] = visibility
	return nil
}

func (listingAssetRepo) CreateFolder(context.Context, database.Executor, Folder) error { return nil }
func (listingAssetRepo) ListFolders(context.Context, database.Executor) ([]Folder, error) {
	return []Folder{{ID: "f-docs", Name: "Docs"}}, nil
}

func adminMediaHandler(t *testing.T, repo *listingAssetRepo) (HTTPHandler, http.Handler) {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, "file:media-admin-"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	handler := HTTPHandler{Service: Service{Assets: repo, Transactions: database.NewTransactor(db)}, SettingsRepository: &adminSettingsRepoFake{}, SettingsCache: NewAdminSettingsCache()}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	return handler, mux
}

func adminRequest(method, path, body string, user *auth.User) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if user != nil {
		request = request.WithContext(auth.WithUser(request.Context(), user))
	}
	return request
}

func decodeEnvelope(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("invalid JSON envelope %s: %v", body, err)
	}
	return decoded
}

func TestAdminMediaListFiltersSortsAndPaginates(t *testing.T) {
	repo := &listingAssetRepo{assets: []Asset{
		{ID: "a3", OriginalFilename: "zeta.pdf", MimeType: "application/pdf", SizeBytes: 300, Status: "ready"},
		{ID: "a2", OriginalFilename: "beta.png", MimeType: "image/png", SizeBytes: 200, Status: "ready", FolderID: "f-docs", Tags: []string{"hero"}},
		{ID: "a1", OriginalFilename: "alpha.png", MimeType: "image/png", SizeBytes: 100, Status: "ready"},
	}}
	_, mux := adminMediaHandler(t, repo)
	editor := &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}

	list := func(query string) (int, map[string]any) {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, adminRequest(http.MethodGet, AdminBasePath+query, "", editor))
		return response.Code, decodeEnvelope(t, response.Body.Bytes())
	}
	status, body := list("?kind=image&sort=filename&limit=1")
	items := body["data"].([]any)
	meta := body["metadata"].(map[string]any)
	if status != http.StatusOK || len(items) != 1 || items[0].(map[string]any)["id"] != "a1" || meta["total"] != float64(2) || meta["has_more"] != true {
		t.Fatalf("image list status=%d body=%v", status, body)
	}
	if status, body := list("?q=hero"); status != http.StatusOK || len(body["data"].([]any)) != 1 || body["data"].([]any)[0].(map[string]any)["url"] != "/media/a2" {
		t.Fatalf("tag search status=%d body=%v", status, body)
	}
	if status, body := list("?folder=f-docs&kind=document"); status != http.StatusOK || len(body["data"].([]any)) != 0 {
		t.Fatalf("folder+kind filter status=%d body=%v", status, body)
	}
	if status, body := list("?sort=nope"); status != http.StatusBadRequest || body["code"] != "bad_request" || body["request_id"] == nil {
		t.Fatalf("invalid sort status=%d body=%v", status, body)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodGet, AdminBasePath, "", &auth.User{ID: "viewer", Active: true, Role: auth.RoleViewer}))
	if response.Code != http.StatusForbidden || decodeEnvelope(t, response.Body.Bytes())["code"] != "forbidden" {
		t.Fatalf("viewer list status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAdminMediaUpdateChangesMetadataAndVisibility(t *testing.T) {
	repo := &listingAssetRepo{assets: []Asset{{ID: "a1", OriginalFilename: "alpha.png", MimeType: "image/png", Status: "ready", Visibility: VisibilityPrivate}}}
	_, mux := adminMediaHandler(t, repo)
	editor := &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodPatch, AdminBasePath+"/a1", `{"folder_id":"f-docs","tags":["hero"],"focal_point":{"x":0.25,"y":0.75},"visibility":"public"}`, editor))
	if response.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", response.Code, response.Body.String())
	}
	data := decodeEnvelope(t, response.Body.Bytes())["data"].(map[string]any)
	if data["folder_id"] != "f-docs" || data["visibility"] != VisibilityPublic || data["focal_x"] != 0.25 {
		t.Fatalf("updated asset=%v", data)
	}
	if stored := repo.metadata["a1"]; stored.FolderID != "f-docs" || len(stored.Tags) != 1 || stored.Focal.Y != 0.75 {
		t.Fatalf("stored metadata=%+v", stored)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodPatch, AdminBasePath+"/a1", `{"visibility":"everyone"}`, editor))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"visibility":"invalid"`) {
		t.Fatalf("invalid visibility status=%d body=%s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodPatch, AdminBasePath+"/missing", `{"visibility":"public"}`, editor))
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing asset status=%d", response.Code)
	}
}

func TestAdminMediaSettingsRoundTripAndValidation(t *testing.T) {
	repo := &listingAssetRepo{assets: []Asset{{ID: "logo", MimeType: "image/png", Status: "ready", SizeBytes: 1024}}}
	handler, mux := adminMediaHandler(t, repo)
	handler.Service.Optimizer = nil
	editor := &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodPut, AdminBasePath+"/settings", `{"hotlink_protection":true,"allowed_referer_hosts":["cdn.example.test","bad host"],"watermark_position":"sideways","watermark_opacity":5}`, editor))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid settings status=%d body=%s", response.Code, response.Body.String())
	}
	fields := decodeEnvelope(t, response.Body.Bytes())["fields"].(map[string]any)
	if fields["allowed_referer_hosts"] != "invalid" || fields["watermark_position"] != "invalid" || fields["watermark_opacity"] != "invalid" {
		t.Fatalf("validation fields=%v", fields)
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodPut, AdminBasePath+"/settings", `{"hotlink_protection":true,"allowed_referer_hosts":["CDN.example.test"],"watermark_enabled":false,"watermark_folders":["f-docs"]}`, editor))
	if response.Code != http.StatusOK {
		t.Fatalf("save settings status=%d body=%s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodGet, AdminBasePath+"/settings", "", editor))
	saved := decodeEnvelope(t, response.Body.Bytes())["data"].(map[string]any)
	if response.Code != http.StatusOK || saved["hotlink_protection"] != true || saved["allowed_referer_hosts"].([]any)[0] != "cdn.example.test" || saved["watermark_position"] != "bottom-right" {
		t.Fatalf("saved settings=%v", saved)
	}
}

func TestAdminMediaCleanupRequiresAdminAndConfirmation(t *testing.T) {
	now := time.Now().UTC()
	garbage := &garbageRepoStub{items: []GarbageAsset{{ID: "old", Filename: "old.png", SizeBytes: 50, CreatedAt: now.Add(-10 * 24 * time.Hour), Status: "ready"}}}
	repo := &listingAssetRepo{}
	handler, _ := adminMediaHandler(t, repo)
	db, err := database.Open(context.Background(), database.SQLite, "file:media-cleanup-"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	handler.Garbage = &GarbageCollector{DB: db, Repository: garbage, References: &referencesStub{sets: []map[string]struct{}{{}}}, BlobStore: &garbageBlobStub{}, Retention: 7 * 24 * time.Hour}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	editor := &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}
	admin := &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodGet, AdminBasePath+"/cleanup", "", editor))
	if response.Code != http.StatusForbidden {
		t.Fatalf("editor cleanup status=%d", response.Code)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodGet, AdminBasePath+"/cleanup", "", admin))
	plan := decodeEnvelope(t, response.Body.Bytes())["data"].(map[string]any)
	if response.Code != http.StatusOK || plan["total_bytes"] != float64(50) || len(plan["items"].([]any)) != 1 {
		t.Fatalf("cleanup plan status=%d body=%s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodPost, AdminBasePath+"/cleanup", `{"asset_ids":["old"]}`, admin))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"confirm":"required"`) {
		t.Fatalf("unconfirmed cleanup status=%d body=%s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, adminRequest(http.MethodPost, AdminBasePath+"/cleanup", `{"asset_ids":["old"],"confirm":true}`, admin))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"deleted":1`) {
		t.Fatalf("confirmed cleanup status=%d body=%s", response.Code, response.Body.String())
	}
}
