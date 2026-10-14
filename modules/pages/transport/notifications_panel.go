package transport

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// adminUnreadNotifications feeds the admin topbar's notification bell: the
// unread count and the latest few entries for its dropdown panel.
func (h Handler) adminUnreadNotifications(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || !editorAllowed(r) {
		http.Error(w, "Bu alan için yetkiniz yok.", http.StatusForbidden)
		return
	}
	if h.Notifications == nil {
		http.Error(w, "Bildirimler şu anda kullanılamıyor.", http.StatusServiceUnavailable)
		return
	}
	count, err := h.Notifications.UnreadCount(r.Context(), h.DB, user.ID)
	if err != nil {
		http.Error(w, "Bildirimler yüklenemedi.", http.StatusServiceUnavailable)
		return
	}
	// The admin topbar's notification panel shows the latest few entries.
	type panelItem struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Message string `json:"message"`
		Href    string `json:"href"`
		Read    bool   `json:"read"`
	}
	items := []panelItem{}
	if queryDB, ok := h.DB.(database.QueryExecutor); ok {
		latest, err := h.Notifications.List(r.Context(), queryDB, user.ID, 5)
		if err != nil {
			http.Error(w, "Bildirimler yüklenemedi.", http.StatusServiceUnavailable)
			return
		}
		for _, item := range latest {
			href := "/admin/notifications"
			if strings.HasPrefix(item.TargetPath, "/") && !strings.HasPrefix(item.TargetPath, "//") {
				href = item.TargetPath
			}
			items = append(items, panelItem{ID: item.ID, Title: item.Title, Message: item.Message, Href: href, Read: item.Read})
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		Unread int         `json:"unread"`
		Items  []panelItem `json:"items"`
	}{Unread: count, Items: items})
}
