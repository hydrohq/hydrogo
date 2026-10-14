package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/erasure"
)

func (h HTTPHandler) changeCredentials(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active || h.Service == nil || h.Transactions == (database.Transactor{}) {
		http.Error(w, "authentication unavailable", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid account update", http.StatusBadRequest)
		return
	}
	newEmail := strings.TrimSpace(r.FormValue("email"))
	newPassword := r.FormValue("new_password")
	if newEmail == user.Email {
		newEmail = ""
	}
	if newEmail == "" && newPassword == "" {
		http.Error(w, "no account changes requested", http.StatusBadRequest)
		return
	}
	if newPassword != "" && newPassword != r.FormValue("confirm_password") {
		http.Error(w, "new passwords do not match", http.StatusBadRequest)
		return
	}
	if newPassword != "" && len([]rune(newPassword)) < 8 {
		http.Error(w, "new password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if newPassword != "" {
		if err := ValidatePasswordStrength(newPassword); err != nil {
			http.Error(w, "Choose a longer, less common password.", http.StatusBadRequest)
			return
		}
	}
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.Service.ChangeCredentials(r.Context(), user.ID, r.FormValue("current_password"), newEmail, newPassword, tx)
	})
	if errors.Is(err, ErrInvalidCredentials) {
		http.Error(w, "current password is incorrect", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrUserAlreadyExists) {
		http.Error(w, "email is already in use", http.StatusConflict)
		return
	}
	if err != nil {
		writeAuthDependencyError(w, "account update failed", err, http.StatusBadRequest)
		return
	}
	if r.FormValue("invalidate_other_sessions") == "true" {
		if err := h.Service.RevokeOtherUserSessions(r.Context(), user.ID, HashSessionToken(sessionToken(r))); err != nil {
			writeAuthDependencyError(w, "account update failed", err, http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.Service.RevokeAllUserSessions(r.Context(), user.ID); err != nil {
			writeAuthDependencyError(w, "account update failed", err, http.StatusInternalServerError)
			return
		}
	}
	if h.Audit != nil {
		h.recordAuthEvent(r, "auth.credentials_changed", user.ID)
	}
	if r.FormValue("invalidate_other_sessions") != "true" {
		ClearSessionCookie(w, h.Secure)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) disableUser(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	actor := UserFromContext(r.Context())
	if !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	targetID := r.PathValue("id")
	if targetID == "" || targetID == actor.ID {
		http.Error(w, "administrator cannot disable the current account", http.StatusConflict)
		return
	}
	if err := h.Service.SetUserActive(r.Context(), targetID, false); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		writeAuthDependencyError(w, "user state update failed", err, http.StatusInternalServerError)
		return
	}
	if h.Audit != nil {
		if err := h.Audit.Record(r.Context(), "auth.user_disabled", targetID); err != nil {
			_ = h.Service.SetUserActive(r.Context(), targetID, true)
			http.Error(w, "audit failed", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) updateUserRole(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actor := UserFromContext(r.Context())
	targetID := r.PathValue("id")
	if targetID == "" || targetID == actor.ID {
		http.Error(w, "administrator role cannot be changed here", http.StatusConflict)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid role update", http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateUserRole(r.Context(), targetID, Role(strings.TrimSpace(r.FormValue("role")))); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		writeAuthDependencyError(w, "user role update failed", err, http.StatusBadRequest)
		return
	}
	h.recordAuthEvent(r, "auth.user_role_changed", targetID)
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) updateUserPermissions(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actor := UserFromContext(r.Context())
	targetID := r.PathValue("id")
	if targetID == "" || targetID == actor.ID {
		http.Error(w, "administrator permissions cannot be changed for the current account", http.StatusConflict)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPermissionFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid permission update", http.StatusBadRequest)
		return
	}
	overrides := make(map[string]bool)
	for _, permission := range h.Service.AvailablePermissions() {
		switch r.FormValue("perm_" + permission) {
		case "allow":
			overrides[permission] = true
		case "deny":
			overrides[permission] = false
		}
	}
	if err := h.Service.ReplaceUserPermissions(r.Context(), targetID, overrides); err != nil {
		writeAuthDependencyError(w, "user permissions update failed", err, http.StatusBadRequest)
		return
	}
	h.recordAuthEvent(r, "auth.user_permissions_changed", targetID)
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) exportUser(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var mfaEnabled bool
	if h.MFA != nil && h.MFADB != nil {
		mfaEnabled, _ = h.MFA.Enabled(r.Context(), h.MFADB, user.ID)
	}

	sessions := []map[string]any{}
	if h.Service != nil {
		items, err := h.Service.ListUserSessions(r.Context(), user.ID)
		if err == nil {
			for _, x := range items {
				sessions = append(sessions, map[string]any{
					"id":           x.ID,
					"user_agent":   x.UserAgent,
					"ip_address":   x.IPAddress,
					"created_at":   x.CreatedAt,
					"last_seen_at": x.LastSeenAt,
					"expires_at":   x.ExpiresAt,
				})
			}
		}
	}

	exportData := map[string]any{
		"user": map[string]any{
			"id":         user.ID,
			"email":      user.Email,
			"role":       user.Role,
			"active":     user.Active,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
		"security": map[string]any{
			"mfa_enabled": mfaEnabled,
		},
		"sessions": sessions,
	}

	w.Header().Set("Cache-Control", "no-store")
	api.WriteJSON(w, http.StatusOK, exportData)
}

func (h HTTPHandler) requestErasure(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || h.Service == nil || h.ErasureLog == nil {
		http.Error(w, "erasure unavailable", http.StatusServiceUnavailable)
		return
	}

	intentIDBytes := make([]byte, 16)
	rand.Read(intentIDBytes)
	intentID := hex.EncodeToString(intentIDBytes)

	editionID := h.EditionID
	if editionID == "" {
		editionID = "default"
	}

	intent := erasure.Intent{
		ID:        intentID,
		EditionID: editionID,
		TargetID:  user.ID,
		Kind:      "user_erasure",
		CreatedAt: time.Now().UTC(),
	}

	if _, err := h.ErasureLog.Append(intent); err != nil {
		writeAuthDependencyError(w, "erasure failed", err, http.StatusInternalServerError)
		return
	}

	if err := h.Service.SetUserActive(r.Context(), user.ID, false); err != nil {
		writeAuthDependencyError(w, "erasure failed", err, http.StatusInternalServerError)
		return
	}

	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "auth.user_erasure_requested", user.ID)
	}

	ClearSessionCookie(w, h.Secure)
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "erasure_requested"})
}
