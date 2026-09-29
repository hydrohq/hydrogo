package transport

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type Handler struct {
	Service       trash.Service
	AfterMutation func(context.Context, trash.Kind, string)
}

func (h Handler) Routes() []router.Route {
	return []router.Route{
		{Pattern: "GET /admin/trash", Access: router.Protected, Handler: http.HandlerFunc(h.list)},
		{Pattern: "POST /admin/trash/{kind}/{id}/delete", Access: router.Protected, Handler: http.HandlerFunc(h.move)},
		{Pattern: "POST /admin/trash/{kind}/{id}/restore", Access: router.Protected, Handler: http.HandlerFunc(h.restore)},
	}
}

func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	if !trashEditor(r) {
		http.Error(w, "You do not have permission to view deleted items.", http.StatusForbidden)
		return
	}
	items, err := h.Service.List(r.Context(), 100)
	if err != nil {
		http.Error(w, "Trash is temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequestWithTrash(w, r, "Çöp kutusu", "İçerik", security.NonceFromContext(r.Context()), "trash"); err != nil {
		return
	}
	if err := designsystem.TrashPage(items, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) move(w http.ResponseWriter, r *http.Request) {
	if !trashEditor(r) {
		http.Error(w, "Bu öğeyi çöp kutusuna taşıma yetkiniz yok.", http.StatusForbidden)
		return
	}
	kind, id := trash.Kind(r.PathValue("kind")), r.PathValue("id")
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
	if !trashEditor(r) {
		http.Error(w, "You do not have permission to restore this item.", http.StatusForbidden)
		return
	}
	kind, id := trash.Kind(r.PathValue("kind")), r.PathValue("id")
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

func trashEditor(r *http.Request) bool {
	return auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentWrite)
}
