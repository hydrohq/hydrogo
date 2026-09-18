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

	_, _ = fmt.Fprintf(w, `<section class="media-overview"><div class="media-dropzone-card"><div class="media-dropzone-wrap" id="media-dropzone"><div class="media-dropzone-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg></div><div class="media-dropzone-text"><strong>Yeni Medya Yükle</strong><small>Dosyaları sürükleyip bırakın veya aşağıdan seçin (JPEG, PNG, WebP, MP4, WebM)</small></div><form class="media-upload-form" id="media-upload-form" method="post" action="/admin/media" enctype="multipart/form-data"><input type="hidden" name="_csrf" value="%s"><input type="file" id="media-file-input" name="file" accept="image/jpeg,image/png,image/webp,video/mp4,video/webm,video/quicktime" required><select name="visibility" aria-label="Görünürlük"><option value="public" selected>Herkese açık</option><option value="private">Özel</option></select><button class="button" type="submit"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg><span>Yükle</span></button></form></div></div><div class="media-toolbar-card"><div class="table-toolbar" style="margin-bottom:0;flex:1;"><div class="search-wrap" style="max-width:320px;"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg><input type="search" id="media-search" placeholder="Dosya adı veya ID ile ara..." autocomplete="off"></div><span class="status-badge" id="media-count-badge">%d asset</span></div></div></section><div class="media-grid" id="media-grid">`, csrf, len(assets))

	for _, asset := range assets {
		preview := `<div class="media-placeholder"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="3" y="3" width="18" height="18" rx="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg><span>Dosya</span></div>`
		if strings.HasPrefix(asset.MimeType, "image/") && asset.Status == "ready" {
			preview = `<img src="/media/` + template.HTMLEscapeString(asset.ID) + `" alt="` + template.HTMLEscapeString(asset.OriginalFilename) + `" loading="lazy">`
		} else if strings.HasPrefix(asset.MimeType, "video/") && asset.Status == "ready" {
			preview = `<video preload="metadata" src="/media/` + template.HTMLEscapeString(asset.ID) + `"></video>`
		}

		visBadge := `<span class="media-badge is-public">Açık</span>`
		if asset.Visibility == VisibilityPrivate {
			visBadge = `<span class="media-badge is-private">Özel</span>`
		}

		metaInfo := formatBytes(asset.SizeBytes)
		if asset.Width > 0 && asset.Height > 0 {
			metaInfo = fmt.Sprintf("%d × %d · %s", asset.Width, asset.Height, metaInfo)
		}

		overlay := fmt.Sprintf(`<div class="media-thumb-overlay"><a href="/media/%s" target="_blank" rel="noopener noreferrer" class="media-overlay-btn" title="Yeni sekmede aç"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path><polyline points="15 3 21 3 21 9"></polyline><line x1="10" x2="21" y1="14" y2="3"></line></svg></a><button type="button" class="media-overlay-btn media-copy-link" data-url="/media/%s" title="Bağlantıyı kopyala"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg></button></div>`, url.PathEscape(asset.ID), url.PathEscape(asset.ID))

		_, _ = fmt.Fprintf(w, `<article class="media-card" data-filename="%s" data-id="%s"><div class="media-thumb-wrap">%s%s%s</div><div class="media-card-body"><strong class="media-title" title="%s">%s</strong><small class="media-sub">%s</small><div class="media-card-actions"><code>%s</code><button class="button button-quiet media-copy" type="button" data-copy="%s" title="ID Kopyala">Kopyala</button></div><form class="media-visibility-form" method="post" action="/admin/media/%s/visibility"><input type="hidden" name="_csrf" value="%s"><select name="visibility" aria-label="Görünürlük"><option value="private" %s>Özel</option><option value="public" %s>Herkese açık</option></select><button class="button button-quiet" type="submit">Güncelle</button></form></div></article>`,
			template.HTMLEscapeString(asset.OriginalFilename),
			template.HTMLEscapeString(asset.ID),
			preview,
			visBadge,
			overlay,
			template.HTMLEscapeString(asset.OriginalFilename),
			template.HTMLEscapeString(asset.OriginalFilename),
			metaInfo,
			template.HTMLEscapeString(asset.ID),
			template.HTMLEscapeString(asset.ID),
			url.PathEscape(asset.ID),
			csrf,
			selectedVisibility(asset.Visibility, VisibilityPrivate),
			selectedVisibility(asset.Visibility, VisibilityPublic),
		)
	}

	if len(assets) == 0 {
		_, _ = fmt.Fprint(w, `<p class="empty" style="grid-column:1/-1;padding:32px;text-align:center;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);">Henüz medya yüklenmedi. Yukarıdaki alandan dosya sürükleyerek veya seçerek yükleyebilirsiniz.</p>`)
	}

	_, _ = fmt.Fprintf(w, `</div><script nonce="%s">(()=>{
const copyHandler = (e) => {
	const copyBtn = e.target.closest('.media-copy');
	if (copyBtn) {
		const text = copyBtn.dataset.copy || '';
		if (navigator.clipboard) {
			navigator.clipboard.writeText(text);
			const orig = copyBtn.textContent;
			copyBtn.textContent = 'Kopyalandı';
			copyBtn.style.color = 'var(--green)';
			setTimeout(() => {
				copyBtn.textContent = orig;
				copyBtn.style.color = '';
			}, 1400);
		}
		return;
	}
	const copyLinkBtn = e.target.closest('.media-copy-link');
	if (copyLinkBtn) {
		const relUrl = copyLinkBtn.dataset.url || '';
		const fullUrl = window.location.origin + relUrl;
		if (navigator.clipboard) {
			navigator.clipboard.writeText(fullUrl);
			const origSvg = copyLinkBtn.innerHTML;
			copyLinkBtn.innerHTML = '<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--green)"><polyline points="20 6 9 17 4 12"></polyline></svg>';
			copyLinkBtn.setAttribute('title', 'Bağlantı kopyalandı');
			setTimeout(() => {
				copyLinkBtn.innerHTML = origSvg;
				copyLinkBtn.setAttribute('title', 'Bağlantıyı kopyala');
			}, 1400);
		}
		return;
	}
};
document.addEventListener('click', copyHandler);


const searchInput = document.getElementById('media-search');
const countBadge = document.getElementById('media-count-badge');
if (searchInput) {
	searchInput.addEventListener('input', () => {
		const q = searchInput.value.toLowerCase().trim();
		let visibleCount = 0;
		document.querySelectorAll('.media-card').forEach(c => {
			const f = (c.dataset.filename || '').toLowerCase();
			const id = (c.dataset.id || '').toLowerCase();
			if (!q || f.includes(q) || id.includes(q)) {
				c.classList.remove('is-hidden');
				visibleCount++;
			} else {
				c.classList.add('is-hidden');
			}
		});
		if (countBadge) {
			countBadge.textContent = visibleCount + ' asset';
		}
	});
}

const dropzone = document.getElementById('media-dropzone');
const fileInput = document.getElementById('media-file-input');
if (dropzone && fileInput) {
	['dragenter', 'dragover'].forEach(name => {
		dropzone.addEventListener(name, (e) => {
			e.preventDefault();
			e.stopPropagation();
			dropzone.classList.add('is-dragover');
		});
	});
	['dragleave', 'drop'].forEach(name => {
		dropzone.addEventListener(name, (e) => {
			e.preventDefault();
			e.stopPropagation();
			dropzone.classList.remove('is-dragover');
		});
	});
	dropzone.addEventListener('drop', (e) => {
		if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
			fileInput.files = e.dataTransfer.files;
		}
	});
}
})();</script>`, nonce)
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
