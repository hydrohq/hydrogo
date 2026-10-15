package media

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/storage"
)

// AdminBasePath is the JSON admin surface for the media library.
const AdminBasePath = "/api/v1/admin/media"

const (
	adminListScanLimit = 500
	adminListMaxLimit  = 100
	adminJSONBodyBytes = 8 << 10
)

// AdminAsset is an asset as the admin lists it: the stored record plus the
// URL a client can load it from.
type AdminAsset struct {
	Asset
	URL string `json:"url"`
}

func (h HTTPHandler) adminRoutes() []HTTPRoute {
	routes := []HTTPRoute{
		{Pattern: "GET " + AdminBasePath, Protected: true, Handler: http.HandlerFunc(h.list)},
		{Pattern: "POST " + AdminBasePath, Protected: true, Handler: http.HandlerFunc(h.upload)},
		{Pattern: "GET " + AdminBasePath + "/folders", Protected: true, Handler: http.HandlerFunc(h.folders)},
		{Pattern: "POST " + AdminBasePath + "/folders", Protected: true, Handler: http.HandlerFunc(h.createFolder)},
		{Pattern: "GET " + AdminBasePath + "/settings", Protected: true, Handler: http.HandlerFunc(h.settings)},
		{Pattern: "PUT " + AdminBasePath + "/settings", Protected: true, Handler: http.HandlerFunc(h.saveSettings)},
		{Pattern: "POST " + AdminBasePath + "/direct-upload", Protected: true, Handler: http.HandlerFunc(h.directUpload)},
		{Pattern: "GET " + AdminBasePath + "/{id}", Protected: true, Handler: http.HandlerFunc(h.get)},
		{Pattern: "PATCH " + AdminBasePath + "/{id}", Protected: true, Handler: http.HandlerFunc(h.update)},
		{Pattern: "GET " + AdminBasePath + "/{id}/url", Protected: true, Handler: http.HandlerFunc(h.url)},
		{Pattern: "POST " + AdminBasePath + "/{id}/ready", Protected: true, Handler: http.HandlerFunc(h.ready)},
		{Pattern: "POST " + AdminBasePath + "/{id}/complete", Protected: true, Handler: http.HandlerFunc(h.completeDirectUpload)},
	}
	if h.Garbage != nil {
		routes = append(routes,
			HTTPRoute{Pattern: "GET " + AdminBasePath + "/cleanup", Protected: true, Handler: http.HandlerFunc(h.cleanupPlan)},
			HTTPRoute{Pattern: "POST " + AdminBasePath + "/cleanup", Protected: true, Handler: http.HandlerFunc(h.cleanupDelete)},
		)
	}
	return routes
}

func (h HTTPHandler) requireEditor(w http.ResponseWriter, r *http.Request) bool {
	if editorAllowed(r) {
		return true
	}
	api.WriteError(w, r, http.StatusForbidden, "media management permission required", nil)
	return false
}

func (h HTTPHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	if user != nil && user.Role == auth.RoleAdmin && editorAllowed(r) {
		return true
	}
	api.WriteError(w, r, http.StatusForbidden, "administrator role required", nil)
	return false
}

func validVisibility(value string) bool {
	return value == VisibilityPublic || value == VisibilityPrivate
}

func (h HTTPHandler) presentAsset(asset Asset) AdminAsset {
	return AdminAsset{Asset: asset, URL: "/media/" + url.PathEscape(asset.ID)}
}

func writeMediaError(w http.ResponseWriter, r *http.Request, err error, message string) {
	status := mediaStatus(err)
	if status == http.StatusInternalServerError {
		message = "request could not be completed"
	}
	api.WriteError(w, r, status, message, nil)
}

type adminListQuery struct {
	folder string
	search string
	kind   string
	sort   string
	limit  int
	offset int
}

func parseAdminListQuery(values url.Values) (adminListQuery, error) {
	query := adminListQuery{folder: strings.TrimSpace(values.Get("folder")), search: strings.ToLower(strings.TrimSpace(values.Get("q"))), kind: values.Get("kind"), sort: values.Get("sort"), limit: 50}
	if len(query.search) > 200 {
		return query, errors.New("search too long")
	}
	switch query.kind {
	case "", "image", "document":
	default:
		return query, errors.New("unknown kind")
	}
	switch query.sort {
	case "", "-created_at", "created_at", "filename", "-filename", "size", "-size":
	default:
		return query, errors.New("unknown sort")
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > adminListMaxLimit {
			return query, errors.New("invalid limit")
		}
		query.limit = limit
	}
	if raw := values.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 || offset > adminListScanLimit {
			return query, errors.New("invalid offset")
		}
		query.offset = offset
	}
	return query, nil
}

func (q adminListQuery) matches(asset Asset) bool {
	if q.folder != "" && asset.FolderID != q.folder {
		return false
	}
	if q.kind == "image" && !strings.HasPrefix(asset.MimeType, "image/") {
		return false
	}
	if q.kind == "document" && strings.HasPrefix(asset.MimeType, "image/") {
		return false
	}
	if q.search == "" {
		return true
	}
	if strings.Contains(strings.ToLower(asset.OriginalFilename), q.search) {
		return true
	}
	for _, tag := range asset.Tags {
		if strings.Contains(strings.ToLower(tag), q.search) {
			return true
		}
	}
	return false
}

func (q adminListQuery) order(assets []Asset) {
	switch q.sort {
	case "filename":
		sort.SliceStable(assets, func(i, j int) bool { return assets[i].OriginalFilename < assets[j].OriginalFilename })
	case "-filename":
		sort.SliceStable(assets, func(i, j int) bool { return assets[i].OriginalFilename > assets[j].OriginalFilename })
	case "size":
		sort.SliceStable(assets, func(i, j int) bool { return assets[i].SizeBytes < assets[j].SizeBytes })
	case "-size":
		sort.SliceStable(assets, func(i, j int) bool { return assets[i].SizeBytes > assets[j].SizeBytes })
	case "created_at":
		for i, j := 0, len(assets)-1; i < j; i, j = i+1, j-1 {
			assets[i], assets[j] = assets[j], assets[i]
		}
	}
}

// list answers the newest assets (the repository already orders by creation
// time, newest first), filtered in memory within a bounded window.
func (h HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	query, err := parseAdminListQuery(r.URL.Query())
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query: "+err.Error(), nil)
		return
	}
	assets, err := h.Service.List(r.Context(), h.DB, adminListScanLimit)
	if err != nil {
		writeMediaError(w, r, err, "media list unavailable")
		return
	}
	matched := make([]Asset, 0, len(assets))
	for _, asset := range assets {
		if query.matches(asset) {
			matched = append(matched, asset)
		}
	}
	query.order(matched)
	total := len(matched)
	if query.offset > total {
		query.offset = total
	}
	end := query.offset + query.limit
	if end > total {
		end = total
	}
	items := make([]AdminAsset, 0, end-query.offset)
	for _, asset := range matched[query.offset:end] {
		items = append(items, h.presentAsset(asset))
	}
	api.WriteList(w, items, api.PaginationMeta{Limit: query.limit, Offset: query.offset, Total: total, HasMore: end < total})
}

func (h HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	asset, err := h.Service.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "media not found", nil)
		return
	}
	api.WriteData(w, http.StatusOK, h.presentAsset(asset), nil)
}

type focalPointInput struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type assetUpdateRequest struct {
	FocalPoint *focalPointInput `json:"focal_point"`
	FolderID   *string          `json:"folder_id"`
	Tags       *[]string        `json:"tags"`
	Visibility *string          `json:"visibility"`
}

// update changes metadata (focal point, folder, tags) and/or visibility.
// Omitted fields keep their stored values.
func (h HTTPHandler) update(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	var input assetUpdateRequest
	if !api.DecodeJSON(w, r, &input, adminJSONBodyBytes) {
		return
	}
	id := r.PathValue("id")
	current, err := h.Service.Get(r.Context(), h.DB, id)
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "media not found", nil)
		return
	}
	if input.Visibility != nil && !validVisibility(*input.Visibility) {
		api.WriteValidation(w, r, map[string]string{"visibility": "invalid"})
		return
	}
	if input.FocalPoint != nil || input.FolderID != nil || input.Tags != nil {
		metadata := AssetMetadata{Focal: FocalPoint{X: current.FocalX, Y: current.FocalY}, FolderID: current.FolderID, Tags: current.Tags}
		if input.FocalPoint != nil {
			metadata.Focal = FocalPoint{X: input.FocalPoint.X, Y: input.FocalPoint.Y}
		}
		if input.FolderID != nil {
			metadata.FolderID = strings.TrimSpace(*input.FolderID)
		}
		if input.Tags != nil {
			metadata.Tags = *input.Tags
		}
		if err := h.Service.SetMetadata(r.Context(), id, metadata); err != nil {
			writeMediaError(w, r, err, "media metadata update failed")
			return
		}
	}
	if input.Visibility != nil {
		if !validVisibility(*input.Visibility) {
			api.WriteValidation(w, r, map[string]string{"visibility": "invalid"})
			return
		}
		if err := h.Service.SetVisibility(r.Context(), id, *input.Visibility); err != nil {
			writeMediaError(w, r, err, "media visibility update failed")
			return
		}
	}
	updated, err := h.Service.Get(r.Context(), h.DB, id)
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "media not found", nil)
		return
	}
	api.WriteData(w, http.StatusOK, h.presentAsset(updated), nil)
}

func (h HTTPHandler) folders(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	folders, err := h.Service.ListFolders(r.Context(), h.DB)
	if err != nil {
		writeMediaError(w, r, err, "media folders unavailable")
		return
	}
	if folders == nil {
		folders = []Folder{}
	}
	api.WriteData(w, http.StatusOK, folders, nil)
}

func (h HTTPHandler) createFolder(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	var input struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id,omitempty"`
	}
	if !api.DecodeJSON(w, r, &input, 4096) {
		return
	}
	folder, err := NewFolder(input.Name, input.ParentID)
	if err != nil {
		api.WriteValidation(w, r, map[string]string{"name": "invalid"})
		return
	}
	if err = h.Service.CreateFolder(r.Context(), folder); err != nil {
		if errors.Is(err, ErrInvalidFolder) {
			api.WriteValidation(w, r, map[string]string{"parent_id": "invalid"})
			return
		}
		writeMediaError(w, r, err, "folder could not be created")
		return
	}
	api.WriteData(w, http.StatusCreated, folder, nil)
}

func (h HTTPHandler) url(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	asset, err := h.Service.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil || asset.Status != "ready" {
		api.WriteError(w, r, http.StatusNotFound, "media not found", nil)
		return
	}
	provider, ok := h.Service.BlobStore.(SignedURLProvider)
	if !ok {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media URL unavailable", nil)
		return
	}
	expiry := 5 * time.Minute
	if raw := r.URL.Query().Get("expires"); raw != "" {
		seconds, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || seconds < 1 || seconds > int64((24*time.Hour)/time.Second) {
			api.WriteError(w, r, http.StatusBadRequest, "invalid URL expiry", nil)
			return
		}
		expiry = time.Duration(seconds) * time.Second
	}
	signed, err := provider.PresignedURL(r.Context(), asset.StorageKey, expiry)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media URL unavailable", nil)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"asset_id": asset.ID, "url": signed, "expires_in": int64(expiry / time.Second)}, nil)
}

// upload accepts a multipart "file" plus optional "folder_id", "visibility"
// and "ready" fields. Without ready=true the asset stays pending until
// POST {id}/ready, which lets a client inspect the processed result first.
func (h HTTPHandler) upload(w http.ResponseWriter, r *http.Request) {
	if h.Service.Uploader == nil || guards.IsNil(h.Service.Assets) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media unavailable", nil)
		return
	}
	if !h.requireEditor(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, multipartBodyLimit(h.Service.Uploader.MaxBytes))
	requestFile, err := readMultipartUpload(r, h.Service.Uploader.StorageDir, h.Service.Uploader.MaxBytes)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid media upload", nil)
		return
	}
	defer os.Remove(requestFile.Path)
	file, err := os.Open(requestFile.Path)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid media upload", nil)
		return
	}
	defer file.Close()
	result, err := h.Service.Upload(r.Context(), file, requestFile.Filename)
	if err != nil {
		if errors.Is(err, storage.ErrQuotaExceeded) {
			api.WriteError(w, r, http.StatusInsufficientStorage, "storage limit reached", nil)
			return
		}
		writeMediaError(w, r, err, "media upload failed")
		return
	}
	status := "pending"
	if folderID := strings.TrimSpace(requestFile.Fields["folder_id"]); folderID != "" {
		if err := h.Service.SetMetadata(r.Context(), result.ID, AssetMetadata{Focal: FocalPoint{X: .5, Y: .5}, FolderID: folderID}); err != nil {
			writeMediaError(w, r, err, "media upload failed")
			return
		}
	}
	visibility := requestFile.Fields["visibility"]
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if !validVisibility(visibility) {
		api.WriteValidation(w, r, map[string]string{"visibility": "invalid"})
		return
	}
	if requestFile.Fields["ready"] == "true" {
		if err := h.Service.MarkReady(r.Context(), result.ID, visibility); err != nil {
			writeMediaError(w, r, err, "media upload failed")
			return
		}
		status = "ready"
	}
	code := http.StatusAccepted
	if status == "ready" {
		code = http.StatusCreated
	}
	api.WriteData(w, code, map[string]any{"asset_id": result.ID, "status": status, "visibility": visibility, "mime_type": result.MimeType, "size_bytes": result.SizeBytes, "url": "/media/" + url.PathEscape(result.ID)}, nil)
}

func (h HTTPHandler) ready(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	var input struct {
		Visibility string `json:"visibility"`
	}
	if !api.DecodeJSON(w, r, &input, adminJSONBodyBytes) {
		return
	}
	if input.Visibility == "" {
		input.Visibility = VisibilityPrivate
	}
	if !validVisibility(input.Visibility) {
		api.WriteValidation(w, r, map[string]string{"visibility": "invalid"})
		return
	}
	if err := h.Service.MarkReady(r.Context(), r.PathValue("id"), input.Visibility); err != nil {
		writeMediaError(w, r, err, "media state update failed")
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"asset_id": r.PathValue("id"), "status": "ready", "visibility": input.Visibility}, nil)
}

func (h HTTPHandler) directUpload(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	var input struct {
		ExpiresInSeconds int `json:"expires_in"`
	}
	if r.ContentLength != 0 && !api.DecodeJSON(w, r, &input, adminJSONBodyBytes) {
		return
	}
	expiry := 5 * time.Minute
	if input.ExpiresInSeconds > 0 {
		expiry = time.Duration(input.ExpiresInSeconds) * time.Second
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
	api.WriteData(w, http.StatusCreated, map[string]any{"object_key": intent.ObjectKey, "url": intent.URL, "expires_at": intent.ExpiresAt}, nil)
}

func (h HTTPHandler) completeDirectUpload(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	var input struct {
		ObjectKey  string `json:"object_key"`
		SizeBytes  int64  `json:"size_bytes"`
		SHA256     string `json:"sha256"`
		Visibility string `json:"visibility"`
	}
	if !api.DecodeJSON(w, r, &input, adminJSONBodyBytes) {
		return
	}
	if input.Visibility == "" {
		input.Visibility = VisibilityPrivate
	}
	if !validVisibility(input.Visibility) {
		api.WriteValidation(w, r, map[string]string{"visibility": "invalid"})
		return
	}
	if err := h.Service.CompleteQuarantineUpload(r.Context(), r.PathValue("id"), input.ObjectKey, input.SizeBytes, input.SHA256, input.Visibility); err != nil {
		if errors.Is(err, ErrQuarantineCompletion) {
			api.WriteValidation(w, r, map[string]string{"object_key": "invalid"})
			return
		}
		writeMediaError(w, r, err, "direct upload could not be completed")
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"asset_id": r.PathValue("id"), "status": "ready", "visibility": input.Visibility}, nil)
}

func (h HTTPHandler) settings(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	settings, err := h.adminSettings(r.Context())
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media settings unavailable", nil)
		return
	}
	api.WriteData(w, http.StatusOK, settings, nil)
}

func (h HTTPHandler) saveSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireEditor(w, r) {
		return
	}
	if h.SettingsRepository == nil || h.SettingsCache == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media settings unavailable", nil)
		return
	}
	var settings AdminSettings
	if !api.DecodeJSON(w, r, &settings, adminJSONBodyBytes) {
		return
	}
	if fields := h.validateSettings(r, &settings); len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	if settings.WatermarkEnabled && h.Service.Optimizer == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "image watermark processing is unavailable", nil)
		return
	}
	if err := h.SettingsRepository.SaveAdminSettings(r.Context(), h.DB, settings); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media settings could not be saved", nil)
		return
	}
	if err := h.SettingsCache.Store(r.Context(), h.RequireTenantScope, settings); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media settings could not be saved", nil)
		return
	}
	api.WriteData(w, http.StatusOK, settings, nil)
}

func (h HTTPHandler) validateSettings(r *http.Request, settings *AdminSettings) map[string]string {
	fields := map[string]string{}
	settings.WatermarkText = strings.TrimSpace(settings.WatermarkText)
	settings.WatermarkLogoID = strings.TrimSpace(settings.WatermarkLogoID)
	if settings.WatermarkPosition == "" {
		settings.WatermarkPosition = "bottom-right"
	}
	if settings.WatermarkOpacity == 0 {
		settings.WatermarkOpacity = .6
	}
	if len(settings.WatermarkText) > 128 {
		fields["watermark_text"] = "invalid"
	}
	if settings.WatermarkOpacity < .1 || settings.WatermarkOpacity > 1 {
		fields["watermark_opacity"] = "invalid"
	}
	if settings.WatermarkMargin < 0 || settings.WatermarkMargin > 1000 {
		fields["watermark_margin"] = "invalid"
	}
	if !validWatermarkPosition(settings.WatermarkPosition) {
		fields["watermark_position"] = "invalid"
	}
	if settings.WatermarkEnabled && settings.WatermarkText == "" && settings.WatermarkLogoID == "" {
		fields["watermark_text"] = "required"
	}
	if settings.WatermarkLogoID != "" {
		logo, err := h.Service.Get(r.Context(), h.DB, settings.WatermarkLogoID)
		if err != nil || logo.Status != "ready" || logo.MimeType != "image/png" || logo.SizeBytes > 2<<20 {
			fields["watermark_logo_id"] = "invalid"
		}
	}
	if len(settings.WatermarkFolders) > 32 {
		fields["watermark_folders"] = "invalid"
	} else if len(settings.WatermarkFolders) > 0 {
		folders, err := h.Service.ListFolders(r.Context(), h.DB)
		known := make(map[string]bool, len(folders))
		for _, folder := range folders {
			known[folder.ID] = true
		}
		for _, folderID := range settings.WatermarkFolders {
			if err != nil || !known[folderID] {
				fields["watermark_folders"] = "invalid"
				break
			}
		}
	}
	if len(settings.AllowedRefererHosts) > 32 {
		fields["allowed_referer_hosts"] = "invalid"
	}
	for i, host := range settings.AllowedRefererHosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if !validRefererHost(host) {
			fields["allowed_referer_hosts"] = "invalid"
			break
		}
		settings.AllowedRefererHosts[i] = host
	}
	return fields
}

func validWatermarkPosition(value string) bool {
	return value == "bottom-right" || value == "bottom-left" || value == "top-right" || value == "center"
}

func validRefererHost(host string) bool {
	if host == "" || strings.ContainsAny(host, "/\\:@") {
		return false
	}
	parsed, err := url.Parse("https://" + host)
	return err == nil && parsed.Hostname() != "" && parsed.Host == host
}

// cleanupPlan lists assets nothing references any more and that are older
// than the retention window; cleanupDelete removes a confirmed selection.
func (h HTTPHandler) cleanupPlan(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.Garbage.Plan(r.Context())
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media cleanup unavailable", nil)
		return
	}
	var size int64
	for _, item := range items {
		size += item.SizeBytes
	}
	if items == nil {
		items = []GarbageAsset{}
	}
	api.WriteData(w, http.StatusOK, map[string]any{"items": items, "total_bytes": size}, nil)
}

func (h HTTPHandler) cleanupDelete(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var input struct {
		AssetIDs []string `json:"asset_ids"`
		Confirm  bool     `json:"confirm"`
	}
	if !api.DecodeJSON(w, r, &input, 16<<10) {
		return
	}
	if !input.Confirm {
		api.WriteValidation(w, r, map[string]string{"confirm": "required"})
		return
	}
	deleted, err := h.Garbage.Delete(r.Context(), input.AssetIDs)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "media cleanup could not be completed", nil)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"deleted": deleted}, nil)
}
