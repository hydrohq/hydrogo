package transport

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
)

type Handler struct {
	Service       trash.Service
	AfterMutation func(context.Context, trash.Kind, string)
}

func (h Handler) Routes() []router.Route {
	return []router.Route{
		{Pattern: "POST /admin/trash/{kind}/{id}/delete", Access: router.Protected, Handler: http.HandlerFunc(h.move)},
		{Pattern: "POST /admin/trash/{kind}/{id}/restore", Access: router.Protected, Handler: http.HandlerFunc(h.restore)},
	}
}

func (h Handler) move(w http.ResponseWriter, r *http.Request) {
	kind, id := trash.Kind(r.PathValue("kind")), r.PathValue("id")
	if !trashEditor(r, kind) {
		http.Error(w, "Bu öğeyi çöp kutusuna taşıma yetkiniz yok.", http.StatusForbidden)
		return
	}
	err := h.Service.Move(r.Context(), kind, id, time.Now())
	if errors.Is(err, trash.ErrNotFound) {
		http.Error(w, "Bu öğe bulunamadı.", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Öğe çöp kutusuna taşınamadı. Yeniden deneyin.", http.StatusServiceUnavailable)
		return
	}
	if h.AfterMutation != nil {
		h.AfterMutation(r.Context(), kind, id)
	}
	http.Redirect(w, r, "/admin/trash", http.StatusSeeOther)
}

func (h Handler) restore(w http.ResponseWriter, r *http.Request) {
	kind, id := trash.Kind(r.PathValue("kind")), r.PathValue("id")
	if !trashEditor(r, kind) {
		http.Error(w, "You do not have permission to restore this item.", http.StatusForbidden)
		return
	}
	err := h.Service.Restore(r.Context(), kind, id, time.Now())
	if errors.Is(err, trash.ErrNotFound) {
		http.Error(w, "This item is no longer available to restore.", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "The item could not be restored. Try again.", http.StatusServiceUnavailable)
		return
	}
	if h.AfterMutation != nil {
		h.AfterMutation(r.Context(), kind, id)
	}
	http.Redirect(w, r, "/admin/trash", http.StatusSeeOther)
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
