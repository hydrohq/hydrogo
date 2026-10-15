package transport

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
)

// AdminBasePath is the JSON admin surface for the recycle bin.
const AdminBasePath = "/api/v1/admin/trash"

type Handler struct {
	Service       trash.Service
	AfterMutation func(context.Context, trash.Kind, string)
}

func (h Handler) Routes() []router.Route {
	return []router.Route{
		{Pattern: "GET " + AdminBasePath, Access: router.Protected, Handler: http.HandlerFunc(h.list)},
		{Pattern: "DELETE " + AdminBasePath + "/{kind}/{id}", Access: router.Protected, Handler: http.HandlerFunc(h.move)},
		{Pattern: "POST " + AdminBasePath + "/{kind}/{id}/restore", Access: router.Protected, Handler: http.HandlerFunc(h.restore)},
	}
}

type trashEntry struct {
	Kind      trash.Kind `json:"kind"`
	ID        string     `json:"id"`
	Label     string     `json:"label"`
	DeletedAt time.Time  `json:"deleted_at"`
}

// list answers the trashed entries the caller may manage; ?kind= narrows
// the result to one content kind.
func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !auth.Can(user, auth.PermissionContentRead) && !auth.Can(user, auth.PermissionSiteManage) {
		api.WriteError(w, r, http.StatusForbidden, "content permission required", nil)
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			api.WriteError(w, r, http.StatusBadRequest, "invalid limit", map[string]string{"limit": "invalid"})
			return
		}
		limit = parsed
	}
	kind := trash.Kind(r.URL.Query().Get("kind"))
	entries, err := h.Service.List(r.Context(), limit)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "trash unavailable", nil)
		return
	}
	items := make([]trashEntry, 0, len(entries))
	for _, entry := range entries {
		if kind != "" && entry.Kind != kind {
			continue
		}
		if !trashViewer(user, entry.Kind) {
			continue
		}
		items = append(items, trashEntry{Kind: entry.Kind, ID: entry.ID, Label: entry.Label, DeletedAt: entry.DeletedAt})
	}
	api.WriteList(w, items, api.PaginationMeta{Limit: limit, Total: len(items)})
}

func (h Handler) move(w http.ResponseWriter, r *http.Request) {
	kind, id := trash.Kind(r.PathValue("kind")), r.PathValue("id")
	if !trashEditor(r, kind) {
		api.WriteError(w, r, http.StatusForbidden, "you may not move this item to the trash", nil)
		return
	}
	err := h.Service.Move(r.Context(), kind, id, time.Now())
	if errors.Is(err, trash.ErrNotFound) {
		api.WriteError(w, r, http.StatusNotFound, "item not found", nil)
		return
	}
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the item could not be moved to the trash", nil)
		return
	}
	if h.AfterMutation != nil {
		h.AfterMutation(r.Context(), kind, id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) restore(w http.ResponseWriter, r *http.Request) {
	kind, id := trash.Kind(r.PathValue("kind")), r.PathValue("id")
	if !trashEditor(r, kind) {
		api.WriteError(w, r, http.StatusForbidden, "you may not restore this item", nil)
		return
	}
	err := h.Service.Restore(r.Context(), kind, id, time.Now())
	if errors.Is(err, trash.ErrNotFound) {
		api.WriteError(w, r, http.StatusNotFound, "item is no longer available to restore", nil)
		return
	}
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the item could not be restored", nil)
		return
	}
	if h.AfterMutation != nil {
		h.AfterMutation(r.Context(), kind, id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// trashEditor gates trash mutations per kind: form submissions hold personal
// data collected by site forms, so they require site-level management rights.
func trashEditor(r *http.Request, kind trash.Kind) bool {
	user := auth.UserFromContext(r.Context())
	if kind == trash.Submissions {
		return auth.Can(user, auth.PermissionSiteManage)
	}
	return auth.Can(user, auth.PermissionContentWrite)
}

func trashViewer(user *auth.User, kind trash.Kind) bool {
	if kind == trash.Submissions {
		return auth.Can(user, auth.PermissionSiteManage)
	}
	return auth.Can(user, auth.PermissionContentRead)
}
