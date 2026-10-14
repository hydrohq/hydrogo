package ui

import (
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type NotificationItem struct {
	ID, Title, Message, TargetPath, Created string
	Read                                    bool
}

func RenderBusinessOwnerDashboard(w http.ResponseWriter, r *http.Request, enabled map[string]bool) error {
	return RenderBusinessOwnerDashboardWithSecurityNotice(w, r, enabled, "", "", 0)
}

func RenderBusinessOwnerDashboardWithSecurityNotice(w http.ResponseWriter, r *http.Request, enabled map[string]bool, previousLogin, previousIP string, failedAttempts int) error {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleEditor || !user.Active {
		return errors.New("business dashboard is unavailable")
	}
	if err := AdminStartForRequest(w, r, "Hoş geldiniz", "", "dashboard"); err != nil {
		return err
	}
	if err := AdminLoginSecurityNotice(previousLogin, previousIP, failedAttempts).Render(r.Context(), w); err != nil {
		return err
	}
	if err := BusinessOwnerDashboard(enabled).Render(r.Context(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}
