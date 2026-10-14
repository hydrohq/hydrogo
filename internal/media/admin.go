package media

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

// AdminRoutes expose one reusable asset library for every Studio collection.
func (h HTTPHandler) AdminRoutes() []HTTPRoute {
	routes := []HTTPRoute{
		{Pattern: "GET /admin/media", Protected: true, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "GET /admin/media/static/uploader.js", Protected: true, Handler: http.HandlerFunc(uploaderAsset)},
		{Pattern: "POST /admin/media", Protected: true, Handler: http.HandlerFunc(h.adminUpload)},
		{Pattern: "POST /admin/media/{id}/metadata", Protected: true, Handler: http.HandlerFunc(h.adminMetadata)},
		{Pattern: "POST /admin/media/folders", Protected: true, Handler: http.HandlerFunc(h.adminCreateFolder)},
		{Pattern: "POST /admin/media/settings", Protected: true, Handler: http.HandlerFunc(h.adminSaveSettings)},
		{Pattern: "POST /admin/media/{id}/visibility", Protected: true, Handler: http.HandlerFunc(h.adminVisibility)},
	}
	if h.Garbage != nil {
		routes = append(routes, HTTPRoute{Pattern: "GET /admin/media/cleanup", Protected: true, Handler: http.HandlerFunc(h.adminCleanup)}, HTTPRoute{Pattern: "POST /admin/media/cleanup/delete", Protected: true, Handler: http.HandlerFunc(h.adminDeleteUnused)})
	}
	return routes
}

func (h HTTPHandler) adminCleanup(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleAdmin || !editorAllowed(r) || h.Garbage == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Garbage.Plan(r.Context())
	if err != nil {
		http.Error(w, "media cleanup unavailable", http.StatusServiceUnavailable)
		return
	}
	view := make([]ui.MediaGarbageItem, 0, len(items))
	var size int64
	for _, item := range items {
		view = append(view, ui.MediaGarbageItem{ID: item.ID, Filename: item.Filename, SizeBytes: item.SizeBytes, CreatedAt: item.CreatedAt.UTC().Format("2006-01-02 15:04 UTC")})
		size += item.SizeBytes
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Unused media cleanup", "Media maintenance", "media"); err != nil {
		return
	}
	if err := ui.MediaCleanupPage(view, size, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w); err != nil {
		http.Error(w, "media cleanup unavailable", http.StatusInternalServerError)
		return
	}
	_ = ui.AdminEnd(w)
}

func (h HTTPHandler) adminDeleteUnused(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleAdmin || !editorAllowed(r) || h.Garbage == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || r.FormValue("confirm") != "yes" {
		http.Error(w, "confirm the selected media cleanup", http.StatusBadRequest)
		return
	}
	ids := r.Form["asset_id"]
	deleted, err := h.Garbage.Delete(r.Context(), ids)
	if err != nil {
		http.Error(w, "media cleanup could not be completed", http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = ui.MediaCleanupResult(fmt.Sprintf("Deleted %d unused assets.", deleted)).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/admin/media/cleanup", http.StatusSeeOther)
}

func (h HTTPHandler) adminList(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	assets, err := h.Service.List(r.Context(), h.DB, 200)
	if err != nil {
		http.Error(w, "media library unavailable", http.StatusServiceUnavailable)
		return
	}
	allAssets := append([]Asset(nil), assets...)
	selectedAsset := strings.TrimSpace(r.URL.Query().Get("asset"))
	if len(selectedAsset) > 128 {
		http.Error(w, "invalid asset selection", http.StatusBadRequest)
		return
	}
	if selectedAsset != "" {
		found := false
		for _, asset := range assets {
			if asset.ID == selectedAsset {
				found = true
				break
			}
		}
		if !found {
			if asset, getErr := h.Service.Get(r.Context(), h.DB, selectedAsset); getErr == nil {
				assets = append(assets, asset)
				allAssets = append(allAssets, asset)
			}
		}
	}
	folders, folderErr := h.Service.ListFolders(r.Context(), h.DB)
	if folderErr != nil {
		http.Error(w, "media folders unavailable", http.StatusServiceUnavailable)
		return
	}
	selectedFolder, selectedTag, selectedColor := r.URL.Query().Get("folder"), strings.TrimSpace(r.URL.Query().Get("tag")), strings.TrimSpace(r.URL.Query().Get("color"))
	selectedTags := splitMediaTags(selectedTag)
	if selectedColor != "" && !validHexColor(selectedColor) {
		http.Error(w, "invalid color filter", http.StatusBadRequest)
		return
	}
	filtered := assets[:0]
	for _, asset := range assets {
		if selectedAsset != "" && asset.ID != selectedAsset {
			continue
		}
		if selectedFolder != "" && asset.FolderID != selectedFolder {
			continue
		}
		if len(selectedTags) > 0 {
			assetTags := make(map[string]struct{}, len(asset.Tags))
			for _, tag := range asset.Tags {
				assetTags[strings.ToLower(tag)] = struct{}{}
			}
			// Every requested tag must match.
			matched := true
			for _, tag := range selectedTags {
				if _, ok := assetTags[strings.ToLower(tag)]; !ok {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
		}
		if selectedColor != "" {
			found := false
			for _, color := range asset.DominantColors {
				if strings.EqualFold(color, selectedColor) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, asset)
	}
	assets = filtered
	settings, settingsErr := h.adminSettings(r.Context())
	if settingsErr != nil {
		http.Error(w, "media settings unavailable", http.StatusServiceUnavailable)
		return
	}
	csrf := security.CSRFTokenFromContext(r.Context())
	folderItems := mediaFolderTree(folders)
	formFolders := make([]ui.MediaFolderItem, 0, len(folderItems))
	for _, folder := range folderItems {
		formFolders = append(formFolders, ui.MediaFolderItem{ID: folder.ID, Name: folder.Name, Depth: folder.Depth})
	}
	logos := make([]ui.MediaLogoItem, 0)
	for _, asset := range allAssets {
		if asset.Status == "ready" && asset.MimeType == "image/png" {
			logos = append(logos, ui.MediaLogoItem{ID: asset.ID, Name: asset.OriginalFilename})
		}
	}
	user := auth.UserFromContext(r.Context())
	view := mediaLibraryView{
		CSRF:           csrf,
		SelectedFolder: selectedFolder,
		SelectedTag:    selectedTag,
		Folders:        folderItems,
		ShowCleanup:    user != nil && user.Role == auth.RoleAdmin && h.Garbage != nil,
		Settings:       ui.MediaSettingsFormProps{CSRF: csrf, HotlinkProtection: settings.HotlinkProtection, AllowedRefererHosts: strings.Join(settings.AllowedRefererHosts, ", "), WatermarkEnabled: settings.WatermarkEnabled, WatermarkText: settings.WatermarkText, WatermarkPosition: settings.WatermarkPosition, WatermarkOpacity: strconv.Itoa(int(settings.WatermarkOpacity * 100)), WatermarkMargin: strconv.Itoa(settings.WatermarkMargin), WatermarkLogoID: settings.WatermarkLogoID, WatermarkFolders: formFolders, WatermarkFoldersSelected: append([]string(nil), settings.WatermarkFolders...), WatermarkLogos: logos},
		Cards:          make([]mediaCardView, 0, len(assets)),
	}
	for _, asset := range assets {
		view.Cards = append(view.Cards, h.mediaCard(r, asset, csrf, selectedFolder, formFolders))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Medya", "Medya kütüphanesi", "media"); err != nil {
		return
	}
	if err := mediaLibrary(view).Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}

// mediaCard builds the view of one library card, choosing the preview kind
// from the asset type and whether a PDF cover derivative is ready.
func (h HTTPHandler) mediaCard(r *http.Request, asset Asset, csrf, selectedFolder string, folders []ui.MediaFolderItem) mediaCardView {
	card := mediaCardView{ID: asset.ID, Filename: asset.OriginalFilename, Private: asset.Visibility == VisibilityPrivate, Preview: "file"}
	ready := asset.Status == "ready"
	switch {
	case ready && strings.HasPrefix(asset.MimeType, "image/"):
		card.Preview = "image"
		card.ObjectPosition = (FocalPoint{X: asset.FocalX, Y: asset.FocalY}).ObjectPosition()
	case ready && asset.MimeType == "application/pdf":
		if repo, ok := h.Service.Assets.(DerivativeRepository); ok {
			if _, err := repo.GetReady(r.Context(), h.DB, asset.ID, VariantPDFCover); err == nil {
				card.Preview = "pdf"
			}
		}
	case ready && strings.HasPrefix(asset.MimeType, "video/"):
		card.Preview = "video"
	}
	card.Meta = formatBytes(asset.SizeBytes)
	if asset.Width > 0 && asset.Height > 0 {
		card.Meta = fmt.Sprintf("%d × %d · %s", asset.Width, asset.Height, card.Meta)
	}
	card.Metadata = ui.MediaMetadataFormProps{Action: "/admin/media/" + url.PathEscape(asset.ID) + "/metadata", CSRF: csrf, ReturnFolder: selectedFolder, FolderID: asset.FolderID, Tags: strings.Join(asset.Tags, ", "), FocalX: fmt.Sprintf("%.2f", asset.FocalX), FocalY: fmt.Sprintf("%.2f", asset.FocalY), Folders: folders}
	return card
}

func splitMediaTags(raw string) []string {
	var tags []string
	for _, tag := range strings.Split(raw, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func (h HTTPHandler) adminSaveSettings(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.SettingsRepository == nil {
		http.Error(w, "media settings unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid media settings", http.StatusBadRequest)
		return
	}
	settings := AdminSettings{HotlinkProtection: r.FormValue("hotlink_protection") == "on", WatermarkEnabled: r.FormValue("watermark_enabled") == "on", WatermarkText: strings.TrimSpace(r.FormValue("watermark_text")), WatermarkPosition: r.FormValue("watermark_position"), WatermarkLogoID: strings.TrimSpace(r.FormValue("watermark_logo_id"))}
	if settings.WatermarkEnabled && h.Service.Optimizer == nil {
		http.Error(w, "image watermark processing is unavailable", http.StatusServiceUnavailable)
		return
	}
	if len(settings.WatermarkText) > 128 {
		http.Error(w, "invalid media settings", http.StatusBadRequest)
		return
	}
	if settings.WatermarkPosition == "" {
		settings.WatermarkPosition = "bottom-right"
	}
	opacity, errOpacity := strconv.Atoi(r.FormValue("watermark_opacity"))
	margin, errMargin := strconv.Atoi(r.FormValue("watermark_margin"))
	if errOpacity != nil || opacity < 10 || opacity > 100 || errMargin != nil || margin < 0 || margin > 1000 || !validWatermarkPosition(settings.WatermarkPosition) || settings.WatermarkEnabled && settings.WatermarkText == "" && settings.WatermarkLogoID == "" {
		http.Error(w, "invalid media settings", http.StatusBadRequest)
		return
	}
	if settings.WatermarkLogoID != "" {
		logo, err := h.Service.Get(r.Context(), h.DB, settings.WatermarkLogoID)
		if err != nil || logo.Status != "ready" || logo.MimeType != "image/png" || logo.SizeBytes > 2<<20 {
			http.Error(w, "select a ready PNG logo smaller than 2 MB", http.StatusBadRequest)
			return
		}
	}
	folders, err := h.Service.ListFolders(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "media settings unavailable", http.StatusServiceUnavailable)
		return
	}
	knownFolders := make(map[string]bool, len(folders))
	for _, folder := range folders {
		knownFolders[folder.ID] = true
	}
	for _, folderID := range r.Form["watermark_folders"] {
		if !knownFolders[folderID] || len(settings.WatermarkFolders) >= 32 {
			http.Error(w, "invalid media settings", http.StatusBadRequest)
			return
		}
		settings.WatermarkFolders = append(settings.WatermarkFolders, folderID)
	}
	settings.WatermarkOpacity, settings.WatermarkMargin = float64(opacity)/100, margin
	for _, host := range splitMediaTags(r.FormValue("allowed_referer_hosts")) {
		if !validRefererHost(host) || len(settings.AllowedRefererHosts) >= 32 {
			http.Error(w, "invalid media settings", http.StatusBadRequest)
			return
		}
		settings.AllowedRefererHosts = append(settings.AllowedRefererHosts, strings.ToLower(host))
	}
	if err := h.SettingsRepository.SaveAdminSettings(r.Context(), h.DB, settings); err != nil {
		http.Error(w, "media settings could not be saved", http.StatusServiceUnavailable)
		return
	}
	if err := h.SettingsCache.Store(r.Context(), h.RequireTenantScope, settings); err != nil {
		http.Error(w, "media settings could not be saved", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
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

func (h HTTPHandler) adminMetadata(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid media metadata", http.StatusBadRequest)
		return
	}
	x, errX := strconv.ParseFloat(r.FormValue("focal_x"), 64)
	y, errY := strconv.ParseFloat(r.FormValue("focal_y"), 64)
	if errX != nil || errY != nil {
		http.Error(w, "invalid media metadata", http.StatusBadRequest)
		return
	}
	var tags []string
	for _, tag := range strings.Split(r.FormValue("tags"), ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	err := h.Service.SetMetadata(r.Context(), r.PathValue("id"), AssetMetadata{Focal: FocalPoint{X: x, Y: y}, FolderID: strings.TrimSpace(r.FormValue("folder_id")), Tags: tags})
	if err != nil {
		http.Error(w, "media metadata update failed", mediaStatus(err))
		return
	}
	target := "/admin/media"
	if folder := r.FormValue("return_folder"); folder != "" {
		target += "?folder=" + url.QueryEscape(folder)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h HTTPHandler) adminCreateFolder(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid folder", http.StatusBadRequest)
		return
	}
	parent := strings.TrimSpace(r.FormValue("parent_id"))
	folder, err := NewFolder(r.FormValue("name"), parent)
	if err != nil {
		http.Error(w, "invalid folder", http.StatusBadRequest)
		return
	}
	if err = h.Service.CreateFolder(r.Context(), folder); err != nil {
		http.Error(w, "folder could not be created", mediaStatus(err))
		return
	}
	target := "/admin/media"
	if parent != "" {
		target += "?folder=" + url.QueryEscape(parent)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h HTTPHandler) adminUpload(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.Service.Uploader == nil {
		http.Error(w, "media unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, multipartBodyLimit(h.Service.Uploader.MaxBytes))
	requestFile, err := readMultipartUpload(r, h.Service.Uploader.StorageDir, h.Service.Uploader.MaxBytes)
	if err != nil {
		http.Error(w, "invalid media upload", http.StatusBadRequest)
		return
	}
	defer os.Remove(requestFile.Path)
	file, err := os.Open(requestFile.Path)
	if err != nil {
		http.Error(w, "invalid media upload", http.StatusBadRequest)
		return
	}
	defer file.Close()
	result, err := h.Service.Upload(r.Context(), file, requestFile.Filename)
	if err == nil {
		if folderID := strings.TrimSpace(requestFile.Fields["folder_id"]); folderID != "" {
			err = h.Service.SetMetadata(r.Context(), result.ID, AssetMetadata{Focal: FocalPoint{X: .5, Y: .5}, FolderID: folderID})
		}
	}
	if err == nil {
		visibility := requestFile.Fields["visibility"]
		if visibility == "" {
			visibility = VisibilityPrivate
		}
		err = h.Service.MarkReady(r.Context(), result.ID, visibility)
	}
	if err != nil {
		http.Error(w, "media upload failed", mediaStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
}

func (h HTTPHandler) adminVisibility(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid media visibility", http.StatusBadRequest)
		return
	}
	if err := h.Service.SetVisibility(r.Context(), r.PathValue("id"), r.FormValue("visibility")); err != nil {
		http.Error(w, "media visibility update failed", mediaStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
}

func mediaFolderTree(folders []Folder) []ui.MediaFolderItem {
	children := make(map[string][]Folder, len(folders))
	known := make(map[string]bool, len(folders))
	for _, folder := range folders {
		known[folder.ID] = true
	}
	for _, folder := range folders {
		parent := folder.ParentID
		if parent != "" && !known[parent] {
			parent = ""
		}
		children[parent] = append(children[parent], folder)
	}
	var items []ui.MediaFolderItem
	visited := make(map[string]bool, len(folders))
	var walk func(string, int)
	walk = func(parent string, depth int) {
		for _, folder := range children[parent] {
			if visited[folder.ID] {
				continue
			}
			visited[folder.ID] = true
			items = append(items, ui.MediaFolderItem{ID: folder.ID, Name: folder.Name, Depth: depth, Href: "/admin/media?folder=" + url.QueryEscape(folder.ID)})
			walk(folder.ID, depth+1)
		}
	}
	walk("", 0)
	for _, folder := range folders {
		if !visited[folder.ID] {
			items = append(items, ui.MediaFolderItem{ID: folder.ID, Name: folder.Name, Href: "/admin/media?folder=" + url.QueryEscape(folder.ID)})
			walk(folder.ID, 1)
		}
	}
	return items
}

func validHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}

func formatBytes(value int64) string {
	if value < 1024 {
		return strconv.FormatInt(value, 10) + " B"
	}
	if value < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(value)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(value)/(1024*1024))
}
