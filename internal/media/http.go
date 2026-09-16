package media

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

type HTTPHandler struct {
	Service Service
	DB      database.Executor
}

type HTTPRoute struct {
	Pattern   string
	Protected bool
	Handler   http.Handler
}

const multipartOverheadBytes int64 = 1 << 20

func (h HTTPHandler) Routes() []HTTPRoute {
	return []HTTPRoute{
		{Pattern: "GET /api/media", Protected: true, Handler: http.HandlerFunc(h.list)},
		{Pattern: "GET /api/media/{id}/url", Protected: true, Handler: http.HandlerFunc(h.url)},
		{Pattern: "POST /api/media", Protected: true, Handler: http.HandlerFunc(h.upload)},
		{Pattern: "POST /api/media/{id}/ready", Protected: true, Handler: http.HandlerFunc(h.ready)},
		{Pattern: "GET /media/{id}", Handler: http.HandlerFunc(h.serve)},
	}
}

func (h HTTPHandler) url(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	asset, err := h.Service.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil || asset.Status != "ready" {
		http.NotFound(w, r)
		return
	}
	provider, ok := h.Service.BlobStore.(SignedURLProvider)
	if !ok {
		http.Error(w, "media URL unavailable", http.StatusServiceUnavailable)
		return
	}
	expiry := 5 * time.Minute
	if raw := r.URL.Query().Get("expires"); raw != "" {
		seconds, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || seconds < 1 || seconds > int64((24*time.Hour)/time.Second) {
			http.Error(w, "invalid URL expiry", http.StatusBadRequest)
			return
		}
		expiry = time.Duration(seconds) * time.Second
	}
	signed, err := provider.PresignedURL(r.Context(), asset.StorageKey, expiry)
	if err != nil {
		http.Error(w, "media URL unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"asset_id": asset.ID, "url": signed, "expires_in": int64(expiry / time.Second)})
}

func (h HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	assets, err := h.Service.List(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "media list unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, assets)
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
	storageKey, mimeType := asset.StorageKey, asset.MimeType
	if kind := r.URL.Query().Get("variant"); kind != "" && h.Service.Assets != nil {
		if repo, ok := h.Service.Assets.(DerivativeRepository); ok {
			if variant, variantErr := repo.GetReady(r.Context(), h.DB, asset.ID, kind); variantErr == nil {
				storageKey, mimeType = variant.StorageKey, variant.MimeType
			}
		}
	}
	if h.Service.BlobStore != nil {
		object, getErr := h.Service.BlobStore.Get(r.Context(), storageKey)
		if getErr != nil {
			http.NotFound(w, r)
			return
		}
		defer object.Close()
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
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, path)
}

func (h HTTPHandler) upload(w http.ResponseWriter, r *http.Request) {
	if h.Service.Uploader == nil || guards.IsNil(h.Service.Assets) {
		http.Error(w, "media unavailable", http.StatusServiceUnavailable)
		return
	}
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, multipartBodyLimit(h.Service.Uploader.MaxBytes))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		http.Error(w, "invalid media upload", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	result, err := h.Service.Upload(r.Context(), file, header.Filename)
	if err != nil {
		http.Error(w, "media upload failed", mediaStatus(err))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"asset_id": result.ID, "status": "pending", "mime_type": result.MimeType, "size_bytes": result.SizeBytes})
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

func (h HTTPHandler) ready(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if err := h.Service.MarkReady(r.Context(), r.PathValue("id"), visibility); err != nil {
		http.Error(w, "media state update failed", mediaStatus(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"asset_id": r.PathValue("id"), "status": "ready", "visibility": visibility})
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && (user.Role == auth.RoleAdmin || user.Role == auth.RoleEditor)
}

func mediaStatus(err error) int {
	if errors.Is(err, ErrFileTooLarge) || errors.Is(err, ErrImageTooLarge) || errors.Is(err, ErrDisallowedMimeType) || errors.Is(err, ErrInvalidImageData) || errors.Is(err, ErrInvalidVisibility) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrUploaderRequired) || errors.Is(err, ErrAssetRepositoryRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
