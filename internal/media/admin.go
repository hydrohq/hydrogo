package media

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

// AdminRoutes expose one reusable asset library for every Studio collection.
func (h HTTPHandler) AdminRoutes() []HTTPRoute {
	return []HTTPRoute{
		{Pattern: "GET /admin/media", Protected: true, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "POST /admin/media", Protected: true, Handler: http.HandlerFunc(h.adminUpload)},
		{Pattern: "POST /admin/media/{id}/visibility", Protected: true, Handler: http.HandlerFunc(h.adminVisibility)},
	}
}

func (h HTTPHandler) adminList(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	assets, err := h.Service.List(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "media library unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Medya", "Medya kütüphanesi", security.NonceFromContext(r.Context()), "media"); err != nil {
		return
	}
	nonce := template.HTMLEscapeString(security.NonceFromContext(r.Context()))
	csrf := template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context()))
	_, _ = fmt.Fprintf(w, `<section class="studio-card media-library"><div class="studio-actions" style="justify-content:space-between;margin-bottom:14px;"><strong>Medya yükle</strong><span class="status">%d asset</span></div><form class="media-upload-form" method="post" action="/admin/media" enctype="multipart/form-data"><input type="hidden" name="_csrf" value="%s"><label>Dosya<input type="file" name="file" accept="image/jpeg,image/png,image/webp,video/mp4,video/webm,video/quicktime" required></label><label>Görünürlük<select name="visibility"><option value="private">Özel</option><option value="public">Herkese açık</option></select></label><button type="submit">Medya yükle</button></form></section><section class="studio-card media-library"><div class="media-grid">`, len(assets), csrf)
	for _, asset := range assets {
		preview := `<div class="media-placeholder">Dosya</div>`
		if strings.HasPrefix(asset.MimeType, "image/") && asset.Status == "ready" {
			preview = `<img src="/media/` + template.HTMLEscapeString(asset.ID) + `" alt="` + template.HTMLEscapeString(asset.OriginalFilename) + `">`
		} else if strings.HasPrefix(asset.MimeType, "video/") && asset.Status == "ready" {
			preview = `<video controls preload="metadata" src="/media/` + template.HTMLEscapeString(asset.ID) + `"></video>`
		}
		_, _ = fmt.Fprintf(w, `<article class="media-card">%s<div class="media-card-body"><strong>%s</strong><small>%s · %s</small><small>%d × %d · %s</small><div class="media-card-actions"><code>%s</code><button class="button button-quiet media-copy" type="button" data-copy="%s">Kimliği kopyala</button></div><form class="media-visibility-form" method="post" action="/admin/media/%s/visibility"><input type="hidden" name="_csrf" value="%s"><label>Görünürlük<select name="visibility"><option value="private" %s>Özel</option><option value="public" %s>Herkese açık</option></select></label><button class="button button-quiet" type="submit">Güncelle</button></form></div></article>`, preview, template.HTMLEscapeString(asset.OriginalFilename), template.HTMLEscapeString(asset.MimeType), template.HTMLEscapeString(asset.Visibility), asset.Width, asset.Height, formatBytes(asset.SizeBytes), template.HTMLEscapeString(asset.ID), template.HTMLEscapeString(asset.ID), url.PathEscape(asset.ID), csrf, selectedVisibility(asset.Visibility, VisibilityPrivate), selectedVisibility(asset.Visibility, VisibilityPublic))
	}
	if len(assets) == 0 {
		_, _ = fmt.Fprint(w, `<p class="empty">Henüz görsel yok. İlk görseli yukarıdaki formdan yükleyin.</p>`)
	}
	_, _ = fmt.Fprintf(w, `</div></section><script nonce="%s">if(!window.__hydroMediaCopyBound){window.__hydroMediaCopyBound=true;document.addEventListener('click',event=>{const button=event.target.closest('.media-copy');if(!button)return;navigator.clipboard?.writeText(button.dataset.copy||'');button.textContent='Kopyalandı';setTimeout(()=>button.textContent='Kimliği kopyala',1500)})}</script>`, nonce)
	_ = ui.AdminEnd(w)
}

func (h HTTPHandler) adminUpload(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) || h.Service.Uploader == nil {
		http.Error(w, "media unavailable", http.StatusServiceUnavailable)
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
	if err == nil {
		visibility := r.FormValue("visibility")
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
	if err := h.Service.SetVisibility(r.Context(), r.PathValue("id"), r.FormValue("visibility")); err != nil {
		http.Error(w, "media visibility update failed", mediaStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
}

func selectedVisibility(value, expected string) string {
	if value == expected {
		return "selected"
	}
	return ""
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
