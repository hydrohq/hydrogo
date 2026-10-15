package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

const maxRevisions = 50

type revisionRepository interface {
	ListRevisions(context.Context, database.Executor, string, int) ([]pages.RevisionSnapshot, error)
}

type revisionSummary struct {
	Revision   int       `json:"revision"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	ActorID    string    `json:"actor_id,omitempty"`
	ActorEmail string    `json:"actor_email,omitempty"`
}

type revisionDetail struct {
	revisionSummary
	Body   string        `json:"body"`
	Blocks []pages.Block `json:"blocks"`
}

func (h Handler) revisionRepository(w http.ResponseWriter, r *http.Request) (revisionRepository, bool) {
	history, ok := h.Content.Pages.(revisionRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "revisions are unavailable", nil)
		return nil, false
	}
	return history, true
}

// GET /api/v1/admin/pages/{id}/revisions?limit=
func (h Handler) adminRevisions(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	history, ok := h.revisionRepository(w, r)
	if !ok {
		return
	}
	limit := maxRevisions
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxRevisions {
			api.WriteError(w, r, http.StatusBadRequest, "limit must be between 1 and 50", nil)
			return
		}
		limit = parsed
	}
	if _, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id")); err != nil {
		writePageReadError(w, r, err)
		return
	}
	revisions, err := history.ListRevisions(r.Context(), h.DB, r.PathValue("id"), limit)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "revisions are unavailable", nil)
		return
	}
	items := make([]revisionSummary, 0, len(revisions))
	for _, revision := range revisions {
		items = append(items, summarizeRevision(revision))
	}
	api.WriteList(w, items, api.PaginationMeta{Limit: limit, HasMore: len(revisions) == limit})
}

func summarizeRevision(revision pages.RevisionSnapshot) revisionSummary {
	return revisionSummary{Revision: revision.Revision, Title: revision.Title, CreatedAt: revision.CreatedAt, ActorID: revision.ActorID, ActorEmail: revision.ActorEmail}
}

func (h Handler) findRevision(w http.ResponseWriter, r *http.Request, history revisionRepository, pageID string) (pages.RevisionSnapshot, bool) {
	wanted, err := strconv.Atoi(r.PathValue("revision"))
	if err != nil || wanted <= 0 {
		api.WriteError(w, r, http.StatusNotFound, "revision not found", nil)
		return pages.RevisionSnapshot{}, false
	}
	revisions, err := history.ListRevisions(r.Context(), h.DB, pageID, maxRevisions)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "revisions are unavailable", nil)
		return pages.RevisionSnapshot{}, false
	}
	for _, revision := range revisions {
		if revision.Revision == wanted {
			return revision, true
		}
	}
	api.WriteError(w, r, http.StatusNotFound, "revision not found", nil)
	return pages.RevisionSnapshot{}, false
}

// GET /api/v1/admin/pages/{id}/revisions/{revision}
func (h Handler) adminRevision(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	history, ok := h.revisionRepository(w, r)
	if !ok {
		return
	}
	current, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	revision, ok := h.findRevision(w, r, history, current.ID)
	if !ok {
		return
	}
	detail := revisionDetail{revisionSummary: summarizeRevision(revision), Body: revision.Body, Blocks: revision.Blocks}
	if detail.Blocks == nil {
		detail.Blocks = []pages.Block{}
	}
	api.WriteData(w, http.StatusOK, map[string]any{"revision": detail, "changes": adminstudio.CompareFields(map[string]string{"title": revision.Title, "body": revision.Body}, map[string]string{"title": current.Title, "body": current.Body})}, nil)
}

// POST /api/v1/admin/pages/{id}/revisions/{revision}/restore {version}
// copies the revision's title, body and blocks into the current draft.
func (h Handler) adminRestoreRevision(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var input versionInput
	if !api.DecodeJSON(w, r, &input, maxJSONBodyBytes) {
		return
	}
	history, ok := h.revisionRepository(w, r)
	if !ok {
		return
	}
	current, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	if input.Version < 1 || input.Version != current.Version {
		api.WriteError(w, r, http.StatusConflict, "the page was changed by someone else; reload and try again", nil)
		return
	}
	revision, ok := h.findRevision(w, r, history, current.ID)
	if !ok {
		return
	}
	current.Title, current.Body, current.Blocks = revision.Title, revision.Body, revision.Blocks
	// Editing always produces a draft revision, whatever the page's status.
	current.Status = pages.StatusDraft
	if err := h.Content.UpdateDraft(r.Context(), current, current.Version); err != nil {
		writePageError(w, r, err)
		return
	}
	h.writePage(w, r, current.ID, http.StatusOK)
}

// POST /api/v1/admin/pages/import (multipart: file, map_title, map_slug,
// map_body, map_locale, action=preview|import). Without a mapping it answers
// the file's headers so the client can build one; with action=preview it
// reports what would be created; with action=import it creates the drafts.
func (h Handler) adminImport(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		api.WriteValidation(w, r, map[string]string{"file": "invalid"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		api.WriteValidation(w, r, map[string]string{"file": "required"})
		return
	}
	defer file.Close()
	rows, err := adminstudio.ParseImportFile(file, header.Filename)
	if err != nil {
		api.WriteValidation(w, r, map[string]string{"file": "unsupported"})
		return
	}
	headers := rows[0]
	if r.FormValue("map_title") == "" || r.FormValue("map_slug") == "" || r.FormValue("map_body") == "" {
		api.WriteData(w, http.StatusOK, map[string]any{"step": "map", "headers": headers, "rows": len(rows) - 1}, nil)
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
			api.WriteValidation(w, r, map[string]string{"map_" + field: "invalid"})
			return
		}
		columns[field] = index
	}
	localeIndex, localeMapped := columns["locale"]
	if columns["slug"] == columns["title"] || columns["slug"] == columns["body"] || columns["title"] == columns["body"] || (localeMapped && (localeIndex == columns["slug"] || localeIndex == columns["title"] || localeIndex == columns["body"])) {
		api.WriteValidation(w, r, map[string]string{"mapping": "columns must differ"})
		return
	}
	repository, ok := h.adminRepository(w, r)
	if !ok {
		return
	}
	existing, err := repository.ListAdmin(r.Context(), h.DB, 5000)
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	seen := map[string]bool{}
	for _, item := range existing {
		seen[canonical.Locale(item.Locale)+"\x00"+item.Slug] = true
	}
	valid, duplicates, invalid := planImport(rows[1:], columns, seen)
	if r.FormValue("action") != "import" {
		api.WriteData(w, http.StatusOK, map[string]any{"step": "preview", "valid": len(valid), "duplicates": duplicates, "invalid": invalid}, nil)
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
	api.WriteData(w, http.StatusOK, map[string]any{"step": "done", "created": created, "skipped": skipped}, nil)
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
