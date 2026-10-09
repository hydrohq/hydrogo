package contract_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteMediaAssetLifecycle(t *testing.T) {
	runMediaAssetLifecycle(t, database.SQLite, filepath.Join(t.TempDir(), "media.sqlite"))
}

func TestPostgresMediaAssetLifecycle(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runMediaAssetLifecycle(t, database.Postgres, dsn)
}

func TestSQLiteAdminMediaMutationRoutesCSRFAndPersistence(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "media-admin-metadata.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	runAdminMediaMutationRoutesContract(t, db, database.SQLite)
}

func TestPostgresAdminMediaMutationRoutesCSRFAndPersistence(t *testing.T) {
	db := openIsolatedPostgres(t, context.Background())
	runAdminMediaMutationRoutesContract(t, db, database.Postgres)
}

func runAdminMediaMutationRoutesContract(t *testing.T, db *sql.DB, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := mediastore.Store{Dialect: dialect, RequireTenantScope: true}
	alpha, err := tenant.NewScope("metadata-tenant-alpha", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := tenant.NewScope("metadata-tenant-beta", "beta")
	if err != nil {
		t.Fatal(err)
	}
	alphaCtx := tenant.WithScope(ctx, alpha)
	betaCtx := tenant.WithScope(ctx, beta)
	asset := media.Asset{ID: "metadata-route-asset", StorageKey: "metadata-route-asset.png", OriginalFilename: "asset.png", MimeType: "image/png", SizeBytes: 12}
	if err := database.NewTransactor(db).Within(alphaCtx, func(tx database.Executor) error { return store.CreatePending(alphaCtx, tx, asset) }); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkReady(alphaCtx, db, asset.ID, media.VisibilityPrivate); err != nil {
		t.Fatal(err)
	}
	folder := media.Folder{ID: "metadata-route-folder", Name: "Editorial"}
	if err := store.CreateFolder(alphaCtx, db, folder); err != nil {
		t.Fatal(err)
	}

	uploader, err := media.NewUploader(filepath.Join(t.TempDir(), "uploads"), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	mediaHandler := media.HTTPHandler{
		DB:                 db,
		Service:            media.Service{Uploader: uploader, Assets: store, Transactions: database.NewTransactor(db)},
		SettingsRepository: store, SettingsCache: media.NewAdminSettingsCache(), RequireTenantScope: true,
	}
	mux := http.NewServeMux()
	for _, route := range mediaHandler.AdminRoutes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	handler := security.CSRFMiddleware(false, nil)(mux)
	handler = tenant.NewHostResolver(map[string]tenant.Scope{
		"alpha.example.test": alpha,
		"beta.example.test":  beta,
	}, true).Middleware(handler)
	adminDenied := &auth.User{ID: "admin-denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: false}}
	serveAs := func(host, token string, user *auth.User) *httptest.ResponseRecorder {
		form := url.Values{
			"_csrf": {token}, "focal_x": {"0.25"}, "focal_y": {"0.75"},
			"folder_id": {folder.ID}, "tags": {"hero, header"},
		}
		request := httptest.NewRequest(http.MethodPost, "/admin/media/metadata-route-asset/metadata", strings.NewReader(form.Encode()))
		request.Host = host
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("X-Tenant-ID", beta.TenantID)
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	serve := func(host, token string) *httptest.ResponseRecorder {
		return serveAs(host, token, &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true})
	}
	invalid := serve("alpha.example.test", "wrong-token")
	if invalid.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF status=%d body=%q", invalid.Code, invalid.Body.String())
	}
	before, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || before.FocalX != 0 || before.FocalY != 0 || before.FolderID != "" || len(before.Tags) != 0 {
		t.Fatalf("invalid CSRF changed metadata: asset=%+v err=%v", before, err)
	}
	deniedMetadata := serveAs("alpha.example.test", "valid-token", adminDenied)
	if deniedMetadata.Code != http.StatusForbidden {
		t.Fatalf("admin media deny metadata status=%d body=%q", deniedMetadata.Code, deniedMetadata.Body.String())
	}
	deniedMetadataState, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || deniedMetadataState.FocalX != 0 || deniedMetadataState.FocalY != 0 || deniedMetadataState.FolderID != "" || len(deniedMetadataState.Tags) != 0 {
		t.Fatalf("admin media deny changed metadata: asset=%+v err=%v", deniedMetadataState, err)
	}
	crossTenant := serve("beta.example.test", "valid-token")
	if crossTenant.Code != http.StatusBadRequest {
		t.Fatalf("cross-tenant mutation status=%d body=%q", crossTenant.Code, crossTenant.Body.String())
	}
	untouched, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || untouched.FocalX != 0 || untouched.FocalY != 0 || untouched.FolderID != "" || len(untouched.Tags) != 0 {
		t.Fatalf("cross-tenant request changed metadata: asset=%+v err=%v", untouched, err)
	}
	valid := serve("alpha.example.test", "valid-token")
	if valid.Code != http.StatusSeeOther {
		t.Fatalf("valid editor mutation status=%d body=%q", valid.Code, valid.Body.String())
	}
	after, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || after.FocalX != .25 || after.FocalY != .75 || after.FolderID != folder.ID || len(after.Tags) != 2 || after.Tags[0] != "hero" || after.Tags[1] != "header" {
		t.Fatalf("persisted metadata=%+v err=%v", after, err)
	}
	if _, err := store.Get(betaCtx, db, asset.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("other tenant media read=%v, want no row", err)
	}

	serveVisibilityAs := func(host, token, visibility string, user *auth.User) *httptest.ResponseRecorder {
		form := url.Values{"_csrf": {token}, "visibility": {visibility}}
		request := httptest.NewRequest(http.MethodPost, "/admin/media/metadata-route-asset/visibility", strings.NewReader(form.Encode()))
		request.Host = host
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	serveVisibility := func(host, token, visibility string) *httptest.ResponseRecorder {
		return serveVisibilityAs(host, token, visibility, &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true})
	}
	invalidVisibility := serveVisibility("alpha.example.test", "wrong-token", media.VisibilityPublic)
	if invalidVisibility.Code != http.StatusForbidden {
		t.Fatalf("invalid visibility CSRF status=%d body=%q", invalidVisibility.Code, invalidVisibility.Body.String())
	}
	unchanged, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || unchanged.Visibility != media.VisibilityPrivate {
		t.Fatalf("invalid visibility CSRF changed asset=%+v err=%v", unchanged, err)
	}
	deniedVisibility := serveVisibilityAs("alpha.example.test", "valid-token", media.VisibilityPublic, adminDenied)
	if deniedVisibility.Code != http.StatusForbidden {
		t.Fatalf("admin media deny visibility status=%d body=%q", deniedVisibility.Code, deniedVisibility.Body.String())
	}
	deniedVisibilityState, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || deniedVisibilityState.Visibility != media.VisibilityPrivate {
		t.Fatalf("admin media deny changed visibility asset=%+v err=%v", deniedVisibilityState, err)
	}
	validVisibility := serveVisibility("alpha.example.test", "valid-token", media.VisibilityPublic)
	if validVisibility.Code != http.StatusSeeOther {
		t.Fatalf("valid editor visibility status=%d body=%q", validVisibility.Code, validVisibility.Body.String())
	}
	visible, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || visible.Visibility != media.VisibilityPublic {
		t.Fatalf("persisted visibility asset=%+v err=%v", visible, err)
	}

	makeUploadRequest := func(token string, user *auth.User) *http.Request {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		for name, value := range map[string]string{"_csrf": token, "folder_id": folder.ID, "visibility": media.VisibilityPrivate} {
			if err := form.WriteField(name, value); err != nil {
				t.Fatal(err)
			}
		}
		file, err := form.CreateFormFile("file", "admin-route-upload.png")
		if err != nil {
			t.Fatal(err)
		}
		var imageBytes bytes.Buffer
		if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(file, &imageBytes); err != nil {
			t.Fatal(err)
		}
		if err := form.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/admin/media", &body)
		request.Host = "alpha.example.test"
		request.Header.Set("Content-Type", form.FormDataContentType())
		request.Header.Set("X-Tenant-ID", beta.TenantID)
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		return request.WithContext(auth.WithUser(request.Context(), user))
	}
	invalidUpload := httptest.NewRecorder()
	handler.ServeHTTP(invalidUpload, makeUploadRequest("wrong-token", &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}))
	if invalidUpload.Code != http.StatusForbidden {
		t.Fatalf("invalid upload CSRF status=%d body=%q", invalidUpload.Code, invalidUpload.Body.String())
	}
	assets, err := store.List(alphaCtx, db, 200)
	if err != nil || len(assets) != 1 {
		t.Fatalf("invalid upload changed database assets=%+v err=%v", assets, err)
	}
	deniedUpload := httptest.NewRecorder()
	handler.ServeHTTP(deniedUpload, makeUploadRequest("valid-token", adminDenied))
	if deniedUpload.Code != http.StatusForbidden {
		t.Fatalf("admin media deny upload status=%d body=%q", deniedUpload.Code, deniedUpload.Body.String())
	}
	assets, err = store.List(alphaCtx, db, 200)
	if err != nil || len(assets) != 1 {
		t.Fatalf("admin media deny changed uploaded assets=%+v err=%v", assets, err)
	}
	validUpload := httptest.NewRecorder()
	handler.ServeHTTP(validUpload, makeUploadRequest("valid-token", &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}))
	if validUpload.Code != http.StatusSeeOther {
		t.Fatalf("valid editor upload status=%d body=%q", validUpload.Code, validUpload.Body.String())
	}
	assets, err = store.List(alphaCtx, db, 200)
	if err != nil || len(assets) != 2 {
		t.Fatalf("uploaded assets=%+v err=%v", assets, err)
	}
	var uploaded *media.Asset
	for i := range assets {
		if assets[i].OriginalFilename == "admin-route-upload.png" {
			uploaded = &assets[i]
			break
		}
	}
	if uploaded == nil || uploaded.Status != "ready" || uploaded.Visibility != media.VisibilityPrivate || uploaded.FolderID != folder.ID || uploaded.MimeType != "image/png" || uploaded.Width != 2 || uploaded.Height != 2 {
		t.Fatalf("uploaded asset=%+v", uploaded)
	}
	if _, err := os.Stat(filepath.Join(uploader.StorageDir, uploaded.ID+".png")); err != nil {
		t.Fatalf("stored upload file is missing: %v", err)
	}

	serveSettingsAs := func(host, token string, user *auth.User) *httptest.ResponseRecorder {
		form := url.Values{
			"_csrf": {token}, "hotlink_protection": {"on"}, "allowed_referer_hosts": {"assets.example.test"},
			"watermark_opacity": {"40"}, "watermark_margin": {"16"},
		}
		request := httptest.NewRequest(http.MethodPost, "/admin/media/settings", strings.NewReader(form.Encode()))
		request.Host = host
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("X-Tenant-ID", beta.TenantID)
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	serveSettings := func(host, token string) *httptest.ResponseRecorder {
		return serveSettingsAs(host, token, &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true})
	}
	invalidSettings := serveSettings("alpha.example.test", "wrong-token")
	if invalidSettings.Code != http.StatusForbidden {
		t.Fatalf("invalid settings CSRF status=%d body=%q", invalidSettings.Code, invalidSettings.Body.String())
	}
	if _, found, err := store.LoadAdminSettings(alphaCtx, db); err != nil || found {
		t.Fatalf("invalid settings CSRF persisted found=%t err=%v", found, err)
	}
	deniedSettings := serveSettingsAs("alpha.example.test", "valid-token", adminDenied)
	if deniedSettings.Code != http.StatusForbidden {
		t.Fatalf("admin media deny settings status=%d body=%q", deniedSettings.Code, deniedSettings.Body.String())
	}
	if _, found, err := store.LoadAdminSettings(alphaCtx, db); err != nil || found {
		t.Fatalf("admin media deny persisted settings found=%t err=%v", found, err)
	}
	validSettings := serveSettings("alpha.example.test", "valid-token")
	if validSettings.Code != http.StatusSeeOther {
		t.Fatalf("valid editor settings status=%d body=%q", validSettings.Code, validSettings.Body.String())
	}
	settings, found, err := store.LoadAdminSettings(alphaCtx, db)
	if err != nil || !found || !settings.HotlinkProtection || len(settings.AllowedRefererHosts) != 1 || settings.AllowedRefererHosts[0] != "assets.example.test" || settings.WatermarkOpacity != .4 || settings.WatermarkMargin != 16 {
		t.Fatalf("persisted settings=%+v found=%t err=%v", settings, found, err)
	}
	if _, found, err := store.LoadAdminSettings(betaCtx, db); err != nil || found {
		t.Fatalf("settings leaked to other tenant found=%t err=%v", found, err)
	}
}

func runMediaAssetLifecycle(t *testing.T, dialect database.Dialect, dsn string) {
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	resetContractData(t, db, dialect)
	store := mediastore.Store{Dialect: dialect}
	asset := media.Asset{ID: "asset-1", StorageKey: "sha256/asset-1.png", OriginalFilename: "logo.png", MimeType: "image/png", SizeBytes: 42, Width: 10, Height: 10}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		return store.CreatePending(ctx, tx, asset)
	}); err != nil {
		t.Fatal(err)
	}
	folder, err := media.NewFolder("Editorial", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateFolder(ctx, db, folder); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if err := store.SetMetadata(ctx, db, asset.ID, media.AssetMetadata{Focal: media.FocalPoint{X: .5, Y: .5}, FolderID: folder.ID}); err != nil {
		t.Fatalf("assign pending upload folder: %v", err)
	}
	if err := store.MarkReady(ctx, db, asset.ID, media.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	var status, visibility string
	if err := db.QueryRowContext(ctx, "SELECT status, visibility FROM media_assets WHERE id = 'asset-1'").Scan(&status, &visibility); err != nil {
		t.Fatal(err)
	}
	if status != "ready" || visibility != media.VisibilityPublic {
		t.Fatalf("asset state = %s/%s, want ready/public", status, visibility)
	}
	if err := store.SetMetadata(ctx, db, asset.ID, media.AssetMetadata{Focal: media.FocalPoint{X: .25, Y: .75}, FolderID: folder.ID, Tags: []string{"hero", "header"}}); err != nil {
		t.Fatalf("move/tag asset: %v", err)
	}
	stored, err := store.Get(ctx, db, asset.ID)
	if err != nil || stored.FolderID != folder.ID || stored.FocalX != .25 || len(stored.Tags) != 2 {
		t.Fatalf("metadata = %+v, err=%v", stored, err)
	}
	settings := media.AdminSettings{HotlinkProtection: true, AllowedRefererHosts: []string{"cdn.example.test"}, WatermarkEnabled: true, WatermarkText: "© HydroGo", WatermarkPosition: "bottom-right", WatermarkOpacity: .5, WatermarkMargin: 12, WatermarkLogoID: "logo-asset", WatermarkFolders: []string{folder.ID}}
	if err := store.SaveAdminSettings(ctx, db, settings); err != nil {
		t.Fatalf("save media settings: %v", err)
	}
	gotSettings, found, err := store.LoadAdminSettings(ctx, db)
	if err != nil || !found || !gotSettings.HotlinkProtection || gotSettings.WatermarkText != settings.WatermarkText || gotSettings.WatermarkLogoID != settings.WatermarkLogoID || len(gotSettings.WatermarkFolders) != 1 {
		t.Fatalf("media settings = %+v found=%v err=%v", gotSettings, found, err)
	}
}
