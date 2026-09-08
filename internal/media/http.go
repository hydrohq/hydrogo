package media

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type HTTPHandler struct {
	Service Service
}

type HTTPRoute struct {
	Pattern   string
	Protected bool
	Handler   http.Handler
}

func (h HTTPHandler) Routes() []HTTPRoute {
	return []HTTPRoute{
		{Pattern: "POST /api/media", Protected: true, Handler: http.HandlerFunc(h.upload)},
		{Pattern: "POST /api/media/{id}/ready", Protected: true, Handler: http.HandlerFunc(h.ready)},
	}
}

func (h HTTPHandler) upload(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.Service.Uploader.MaxBytes+1<<20)
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
	return http.StatusInternalServerError
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
