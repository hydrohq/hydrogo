package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func (h Handler) restoreRevision(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	current, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	history, ok := h.Content.Pages.(interface {
		ListRevisions(context.Context, database.Executor, string, int) ([]pages.RevisionSnapshot, error)
	})
	if !ok {
		http.Error(w, "revision unavailable", http.StatusServiceUnavailable)
		return
	}
	revisions, err := history.ListRevisions(r.Context(), h.DB, current.ID, 50)
	if err != nil {
		http.Error(w, "revision unavailable", http.StatusServiceUnavailable)
		return
	}
	wanted, err := strconv.Atoi(r.PathValue("revision"))
	if err != nil || wanted <= 0 {
		http.Error(w, "revision unavailable", http.StatusBadRequest)
		return
	}
	for _, revision := range revisions {
		if revision.Revision != wanted {
			continue
		}
		current.Title, current.Body, current.Blocks = revision.Title, revision.Body, revision.Blocks
		if err := h.Content.UpdateDraft(r.Context(), current, current.Version); err != nil {
			http.Error(w, pageError(err), pageStatus(err))
			return
		}
		http.Redirect(w, r, "/admin/pages/"+url.PathEscape(current.ID), http.StatusSeeOther)
		return
	}
	http.NotFound(w, r)
}

func (h Handler) adminTranslate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	source, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page unavailable", pageReadStatus(err))
		return
	}
	locale := canonical.Locale(r.FormValue("locale"))
	if locale == "" || locale == source.Locale {
		http.Error(w, "invalid translation", http.StatusBadRequest)
		return
	}
	repository, ok := h.Content.Pages.(adminListRepository)
	if !ok {
		http.Error(w, "page unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "page unavailable", http.StatusServiceUnavailable)
		return
	}
	target := pages.Page{}
	for _, item := range items {
		if item.Locale == locale && item.Slug == source.Slug {
			target, err = h.Content.Get(r.Context(), h.DB, item.ID)
			if err != nil {
				http.Error(w, "page unavailable", pageReadStatus(err))
				return
			}
			break
		}
	}
	target.Locale, target.Slug, target.Title, target.Body = locale, source.Slug, strings.TrimSpace(r.FormValue("title")), r.FormValue("body")
	if target.ID == "" {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			http.Error(w, "translation could not be saved", http.StatusInternalServerError)
			return
		}
		target.ID = "page-" + hex.EncodeToString(id)
		if err := h.Content.CreateDraft(r.Context(), target); err != nil {
			http.Error(w, pageError(err), pageStatus(err))
			return
		}
	} else {
		version, parseErr := strconv.Atoi(r.FormValue("version"))
		if parseErr != nil || version != target.Version {
			http.Error(w, "translation changed; reload and try again", http.StatusConflict)
			return
		}
		if err := h.Content.UpdateDraft(r.Context(), target, version); err != nil {
			http.Error(w, pageError(err), pageStatus(err))
			return
		}
	}
	http.Redirect(w, r, "/admin/pages/"+url.PathEscape(target.ID), http.StatusSeeOther)
}
