package transport

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func (h Handler) adminImport(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "Bu işlem için yetkiniz yok.", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method == http.MethodGet {
		if ui.AdminStartForRequest(w, r, "İçerik içe aktar", "", "pages") != nil {
			return
		}
		_ = ui.ImportWizard(nil, security.CSRFTokenFromContext(r.Context()), false).Render(r.Context(), w)
		_ = ui.AdminEnd(w)
		return
	}
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		http.Error(w, "Dosya okunamadı. CSV veya Excel dosyasını yeniden seçin.", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bir CSV veya Excel dosyası seçin.", http.StatusBadRequest)
		return
	}
	defer file.Close()
	rows, err := adminstudio.ParseImportFile(file, header.Filename)
	if err != nil {
		http.Error(w, "Dosya biçimi okunamadı. CSV veya .xlsx dosyası kullanın.", http.StatusBadRequest)
		return
	}
	headers := rows[0]
	if r.FormValue("map_title") == "" || r.FormValue("map_slug") == "" {
		if ui.AdminStartForRequest(w, r, "Sütunları eşleştir", "İçe aktarma sihirbazı", "pages") != nil {
			return
		}
		_ = ui.ImportWizard(headers, security.CSRFTokenFromContext(r.Context()), true).Render(r.Context(), w)
		_ = ui.AdminEnd(w)
		return
	}
	columns := map[string]int{}
	for _, field := range []string{"locale", "slug", "title", "body"} {
		raw := r.FormValue("map_" + field)
		if raw == "" {
			continue
		}
		index, parseErr := strconv.Atoi(raw)
		if parseErr != nil || index < 0 || index >= len(headers) {
			http.Error(w, "Sütun eşlemesi geçersiz. Lütfen yeniden deneyin.", http.StatusBadRequest)
			return
		}
		columns[field] = index
	}
	if _, ok := columns["body"]; !ok {
		http.Error(w, "İçerik metni için bir sütun seçin.", http.StatusBadRequest)
		return
	}
	localeIndex, localeMapped := columns["locale"]
	if columns["slug"] == columns["title"] || columns["slug"] == columns["body"] || columns["title"] == columns["body"] || (localeMapped && (localeIndex == columns["slug"] || localeIndex == columns["title"] || localeIndex == columns["body"])) {
		http.Error(w, "Başlık, bağlantı adı ve metin için farklı sütunlar seçin.", http.StatusBadRequest)
		return
	}
	mapValues := map[string]string{}
	for name, index := range columns {
		mapValues[name] = strconv.Itoa(index)
	}
	repo, ok := h.Content.Pages.(adminListRepository)
	if !ok {
		http.Error(w, "Aktarım şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	existing, err := repo.ListAdmin(r.Context(), h.DB, 5000)
	if err != nil {
		http.Error(w, "İçerikler yüklenemedi.", http.StatusServiceUnavailable)
		return
	}
	seen := map[string]bool{}
	for _, item := range existing {
		seen[canonical.Locale(item.Locale)+"\x00"+item.Slug] = true
	}
	valid, duplicates, invalid := planImport(rows[1:], columns, seen)
	if r.FormValue("action") != "import" {
		if ui.AdminStartForRequest(w, r, "Aktarım önizlemesi", "İçe aktarma önizlemesi", "pages") != nil {
			return
		}
		_ = ui.ImportPreview(security.CSRFTokenFromContext(r.Context()), mapValues, len(valid), duplicates, invalid).Render(r.Context(), w)
		_ = ui.AdminEnd(w)
		return
	}
	created, skipped := 0, duplicates+invalid
	for _, page := range valid {
		page.ID = newImportID()
		if page.ID == "" || h.Content.CreateDraft(r.Context(), page) != nil {
			skipped++
			continue
		}
		created++
	}
	renderAdminScreen(w, r, "Aktarım tamamlandı", "İçe aktarma sonucu", pageImportResult(created, skipped))
}

func planImport(rows [][]string, columns map[string]int, existing map[string]bool) ([]pages.Page, int, int) {
	valid := make([]pages.Page, 0, len(rows))
	duplicates, invalid := 0, 0
	seen := make(map[string]bool, len(existing))
	for key := range existing {
		seen[key] = true
	}
	for _, row := range rows {
		value := func(name, fallback string) string {
			if i, ok := columns[name]; ok && i >= 0 && i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return fallback
		}
		locale := canonical.Locale(value("locale", "tr"))
		slug, err := canonical.Path(value("slug", ""))
		title, body := value("title", ""), value("body", "")
		if err != nil || locale == "" || len(locale) > 16 || slug == "" || len(slug) > 256 || title == "" || len(title) > 256 || body == "" || len(body) > 1<<20 {
			invalid++
			continue
		}
		key := locale + "\x00" + slug
		if seen[key] {
			duplicates++
			continue
		}
		seen[key] = true
		valid = append(valid, pages.Page{Locale: locale, Slug: slug, Title: title, Body: body, Status: pages.StatusDraft})
	}
	return valid, duplicates, invalid
}

func newImportID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return "page-import-" + hex.EncodeToString(b[:])
}
