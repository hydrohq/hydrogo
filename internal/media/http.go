package media

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type HTTPHandler struct {
	Service             Service
	Garbage             *GarbageCollector
	DB                  database.Executor
	HotlinkProtection   bool
	CanonicalOrigin     string
	AllowedRefererHosts []string
	SettingsRepository  AdminSettingsRepository
	SettingsCache       *AdminSettingsCache
	SettingsDefaults    AdminSettings
	RequireTenantScope  bool
}

type HTTPRoute struct {
	Pattern   string
	Protected bool
	Handler   http.Handler
}

const multipartOverheadBytes int64 = 1 << 20

type mediaSettingsContextKey struct{}

func (h HTTPHandler) Routes() []HTTPRoute {
	serve := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		settings, err := h.adminSettings(r.Context())
		if err != nil {
			http.Error(w, "media settings unavailable", http.StatusServiceUnavailable)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), mediaSettingsContextKey{}, settings))
		h.serve(w, r)
	})
	return append(h.adminRoutes(), HTTPRoute{Pattern: "GET /media/{id}", Handler: serve})
}

func (h HTTPHandler) adminSettings(ctx context.Context) (AdminSettings, error) {
	defaults := h.SettingsDefaults
	if !defaults.HotlinkProtection && h.HotlinkProtection {
		defaults.HotlinkProtection = true
	}
	if len(defaults.AllowedRefererHosts) == 0 {
		defaults.AllowedRefererHosts = append([]string(nil), h.AllowedRefererHosts...)
	}
	settings, err := h.SettingsCache.Load(ctx, h.RequireTenantScope, defaults, h.SettingsRepository, h.DB)
	if settings.WatermarkOpacity == 0 {
		settings.WatermarkOpacity = .6
	}
	if settings.WatermarkPosition == "" {
		settings.WatermarkPosition = "bottom-right"
	}
	return settings, err
}

func (h HTTPHandler) serve(w http.ResponseWriter, r *http.Request) {
	if h.Service.Uploader == nil {
		http.Error(w, "media unavailable", http.StatusServiceUnavailable)
		return
	}
	asset, err := h.Service.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil || asset.Status != "ready" {
		http.NotFound(w, r)
		return
	}
	if asset.Visibility == VisibilityPrivate && !editorAllowed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	settings, _ := r.Context().Value(mediaSettingsContextKey{}).(AdminSettings)
	serveAsset := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, r, asset, settings) }))
	if settings.HotlinkProtection && (strings.HasPrefix(asset.MimeType, "image/") || isDocumentFilename(asset.OriginalFilename)) {
		HotlinkGuard(h.CanonicalOrigin, settings.AllowedRefererHosts, serveAsset).ServeHTTP(w, r)
		return
	}
	serveAsset.ServeHTTP(w, r)
}

func (h HTTPHandler) serveAsset(w http.ResponseWriter, r *http.Request, asset Asset, settings AdminSettings) {
	watermarked := strings.HasPrefix(asset.MimeType, "image/") && settings.WatermarkEnabled && (settings.WatermarkText != "" || settings.WatermarkLogoID != "")
	if watermarked && len(settings.WatermarkFolders) > 0 {
		watermarked = false
		for _, folderID := range settings.WatermarkFolders {
			if folderID == asset.FolderID {
				watermarked = true
				break
			}
		}
	}
	storageKey, mimeType := asset.StorageKey, asset.MimeType
	servedVariant := false
	kind := r.URL.Query().Get("variant")
	if kind == "" && watermarked && !editorAllowed(r) {
		kind = VariantCard
	}
	if kind != "" && h.Service.Assets != nil {
		if repo, ok := h.Service.Assets.(DerivativeRepository); ok {
			if variant, variantErr := repo.GetReady(r.Context(), h.DB, asset.ID, kind); variantErr == nil {
				storageKey, mimeType = variant.StorageKey, variant.MimeType
				servedVariant = true
			} else if watermarked && !editorAllowed(r) {
				w.Header().Set("Retry-After", "3")
				http.Error(w, "image processing in progress", http.StatusServiceUnavailable)
				return
			}
		} else if watermarked && !editorAllowed(r) {
			http.Error(w, "image processing in progress", http.StatusServiceUnavailable)
			return
		}
	}
	if asset.Visibility == VisibilityPrivate && isDocumentFilename(asset.OriginalFilename) && !servedVariant {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeOriginalFilename(asset.OriginalFilename)}))
	}
	if h.Service.BlobStore != nil {
		object, getErr := h.Service.BlobStore.Get(r.Context(), storageKey)
		if getErr != nil {
			http.NotFound(w, r)
			return
		}
		defer object.Close()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", mimeType)
		io.Copy(w, object)
		return
	}
	path := filepath.Join(h.Service.Uploader.StorageDir, filepath.Base(storageKey))
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, path)
}

func multipartBodyLimit(fileLimit int64) int64 {
	if fileLimit <= 0 {
		return multipartOverheadBytes
	}
	const maxInt64 = int64(^uint64(0) >> 1)
	if fileLimit > maxInt64-multipartOverheadBytes {
		return maxInt64
	}
	return fileLimit + multipartOverheadBytes
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionMediaManage)
}

func mediaStatus(err error) int {
	if errors.Is(err, ErrFileTooLarge) || errors.Is(err, ErrImageTooLarge) || errors.Is(err, ErrDisallowedMimeType) || errors.Is(err, ErrInvalidImageData) || errors.Is(err, ErrInvalidVisibility) || errors.Is(err, ErrInvalidFolder) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrUploaderRequired) || errors.Is(err, ErrAssetRepositoryRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}
