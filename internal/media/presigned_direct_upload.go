package media

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func (h HTTPHandler) uploadURL(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	if h.RequireTenantScope {
		if _, err := tenant.Require(r.Context()); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "tenant scope required", nil)
			return
		}
	}
	var input struct {
		Filename         string `json:"filename"`
		MimeType         string `json:"mime_type"`
		SizeBytes        int64  `json:"size_bytes"`
		FolderID         string `json:"folder_id"`
		Visibility       string `json:"visibility"`
		ExpiresInSeconds int    `json:"expires_in"`
	}
	if r.ContentLength != 0 && !api.DecodeJSON(w, r, &input, adminJSONBodyBytes) {
		return
	}
	if input.Filename != "" {
		if err := validateUploadFilename(input.Filename); err != nil {
			api.WriteValidation(w, r, map[string]string{"filename": "disallowed file extension"})
			return
		}
		input.Filename = safeOriginalFilename(input.Filename)
	}
	if input.SizeBytes < 0 {
		api.WriteValidation(w, r, map[string]string{"size_bytes": "invalid size"})
		return
	}
	if h.Service.Uploader != nil && h.Service.Uploader.MaxBytes > 0 && input.SizeBytes > h.Service.Uploader.MaxBytes {
		api.WriteValidation(w, r, map[string]string{"size_bytes": "exceeds maximum allowed size"})
		return
	}
	expiry := 15 * time.Minute
	if input.ExpiresInSeconds > 0 {
		if input.ExpiresInSeconds > 900 {
			expiry = 15 * time.Minute
		} else if input.ExpiresInSeconds < 5 {
			expiry = 5 * time.Second
		} else {
			expiry = time.Duration(input.ExpiresInSeconds) * time.Second
		}
	}
	intent, err := h.Service.PrepareQuarantineUpload(r.Context(), expiry)
	if err != nil {
		if errors.Is(err, ErrDirectUploadUnavailable) {
			api.WriteError(w, r, http.StatusServiceUnavailable, "direct upload unavailable", nil)
			return
		}
		writeMediaError(w, r, err, "direct upload unavailable")
		return
	}
	assetID, err := directUploadID()
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "failed to generate upload intent", nil)
		return
	}
	visibility := input.Visibility
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if !validVisibility(visibility) {
		visibility = VisibilityPrivate
	}
	if !guards.IsNil(h.Service.Assets) && input.Filename != "" {
		pendingAsset := Asset{
			ID:               assetID,
			StorageKey:       intent.ObjectKey,
			OriginalFilename: input.Filename,
			MimeType:         input.MimeType,
			SizeBytes:        input.SizeBytes,
			FolderID:         input.FolderID,
			Visibility:       visibility,
			Status:           "pending",
		}
		_ = h.Service.Transactions.Within(r.Context(), func(tx database.Executor) error {
			return h.Service.Assets.CreatePending(r.Context(), tx, pendingAsset)
		})
	}
	api.WriteData(w, http.StatusCreated, map[string]any{
		"asset_id":   assetID,
		"object_key": intent.ObjectKey,
		"upload_url": intent.URL,
		"url":        intent.URL,
		"expires_at": intent.ExpiresAt,
	}, nil)
}

func (h HTTPHandler) confirmUpload(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	if h.RequireTenantScope {
		if _, err := tenant.Require(r.Context()); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "tenant scope required", nil)
			return
		}
	}
	var input struct {
		AssetID    string `json:"asset_id"`
		ObjectKey  string `json:"object_key"`
		Filename   string `json:"filename"`
		MimeType   string `json:"mime_type"`
		SizeBytes  int64  `json:"size_bytes"`
		SHA256     string `json:"sha256"`
		Visibility string `json:"visibility"`
	}
	if !api.DecodeJSON(w, r, &input, adminJSONBodyBytes) {
		return
	}
	if strings.TrimSpace(input.ObjectKey) == "" {
		api.WriteValidation(w, r, map[string]string{"object_key": "object_key is required"})
		return
	}
	if input.Visibility == "" {
		input.Visibility = VisibilityPrivate
	}
	if !validVisibility(input.Visibility) {
		api.WriteValidation(w, r, map[string]string{"visibility": "invalid"})
		return
	}
	assetID := strings.TrimSpace(input.AssetID)
	if assetID == "" {
		newID, err := directUploadID()
		if err != nil {
			api.WriteError(w, r, http.StatusInternalServerError, "failed to generate asset identifier", nil)
			return
		}
		assetID = newID
		if !guards.IsNil(h.Service.Assets) {
			pendingAsset := Asset{
				ID:               assetID,
				StorageKey:       input.ObjectKey,
				OriginalFilename: safeOriginalFilename(input.Filename),
				MimeType:         input.MimeType,
				SizeBytes:        input.SizeBytes,
				Visibility:       input.Visibility,
				Status:           "pending",
			}
			_ = h.Service.Transactions.Within(r.Context(), func(tx database.Executor) error {
				return h.Service.Assets.CreatePending(r.Context(), tx, pendingAsset)
			})
		}
	}
	if err := h.Service.CompleteQuarantineUpload(r.Context(), assetID, input.ObjectKey, input.SizeBytes, input.SHA256, input.Visibility); err != nil {
		if errors.Is(err, ErrQuarantineCompletion) {
			api.WriteValidation(w, r, map[string]string{"object_key": "invalid"})
			return
		}
		writeMediaError(w, r, err, "direct upload could not be completed")
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{
		"asset_id":   assetID,
		"status":     "ready",
		"visibility": input.Visibility,
		"url":        "/media/" + url.PathEscape(assetID),
	}, nil)
}
