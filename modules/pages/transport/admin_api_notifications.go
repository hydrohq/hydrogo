package transport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type notificationItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Href      string    `json:"href"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type digestPreference struct {
	Enabled   bool   `json:"enabled"`
	Timezone  string `json:"timezone"`
	Frequency string `json:"frequency"`
	Available bool   `json:"available"`
}

func (h Handler) notificationsAllowed(w http.ResponseWriter, r *http.Request) (string, bool) {
	user := currentUser(r)
	if user == nil || !editorAllowed(r) {
		writeForbidden(w, r)
		return "", false
	}
	if h.Notifications == nil || database.RequireExecutor(h.DB) != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "notifications are unavailable", nil)
		return "", false
	}
	return user.ID, true
}

// GET /api/v1/admin/notifications?limit= lists the signed-in editor's
// notifications; metadata carries the unread count and digest preference.
func (h Handler) adminNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.notificationsAllowed(w, r)
	if !ok {
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			api.WriteError(w, r, http.StatusBadRequest, "limit must be between 1 and 100", nil)
			return
		}
		limit = parsed
	}
	queryDB, ok := h.DB.(database.QueryExecutor)
	if !ok {
		api.WriteError(w, r, http.StatusServiceUnavailable, "notifications are unavailable", nil)
		return
	}
	items, err := h.Notifications.List(r.Context(), queryDB, userID, limit)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "notifications could not be loaded", nil)
		return
	}
	unread, err := h.Notifications.UnreadCount(r.Context(), h.DB, userID)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "notifications could not be loaded", nil)
		return
	}
	enabled, timezone, frequency, err := h.Notifications.DigestPreferenceDetailsFor(r.Context(), h.DB, userID)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "notification preferences could not be loaded", nil)
		return
	}
	view := make([]notificationItem, 0, len(items))
	for _, item := range items {
		href := ""
		if strings.HasPrefix(item.TargetPath, "/") && !strings.HasPrefix(item.TargetPath, "//") {
			href = item.TargetPath
		}
		view = append(view, notificationItem{ID: item.ID, Title: item.Title, Message: item.Message, Href: href, Read: item.Read, CreatedAt: item.CreatedAt})
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, view, map[string]any{
		"unread": unread,
		"digest": digestPreference{Enabled: enabled, Timezone: timezone, Frequency: frequency, Available: h.WeeklyDigestEnabled},
	})
}

// POST /api/v1/admin/notifications/read-all
func (h Handler) adminNotificationsReadAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.notificationsAllowed(w, r)
	if !ok {
		return
	}
	if err := h.Notifications.MarkAllRead(r.Context(), h.DB, userID); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "notifications could not be updated", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/admin/notifications/{id}/dismiss
func (h Handler) adminNotificationDismiss(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.notificationsAllowed(w, r)
	if !ok {
		return
	}
	if err := h.Notifications.Dismiss(r.Context(), h.DB, userID, r.PathValue("id")); err != nil {
		if errors.Is(err, adminstudiosql.ErrInvalid) {
			api.WriteError(w, r, http.StatusNotFound, "notification not found", nil)
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "notifications could not be updated", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type digestInput struct {
	Enabled   bool   `json:"enabled"`
	Timezone  string `json:"timezone"`
	Frequency string `json:"frequency"`
}

// PUT /api/v1/admin/notifications/digest {enabled, timezone, frequency}
func (h Handler) adminNotificationDigest(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.notificationsAllowed(w, r)
	if !ok {
		return
	}
	if !h.WeeklyDigestEnabled {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the e-mail digest is not configured", nil)
		return
	}
	var input digestInput
	if !api.DecodeJSON(w, r, &input, 4<<10) {
		return
	}
	if err := h.Notifications.SetDigestPreferenceWithFrequency(r.Context(), h.DB, userID, input.Timezone, input.Frequency, input.Enabled); err != nil {
		if errors.Is(err, adminstudiosql.ErrInvalid) {
			api.WriteValidation(w, r, map[string]string{"timezone": "invalid"})
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "notification preferences could not be saved", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
