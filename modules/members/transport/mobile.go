package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/push"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

const maxMemberMobileJSONBytes = 16 << 10

type memberMobileLoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
}

type memberMobileRegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
}

type memberMobileRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// mobileLogin issues a member-only token pair for consumer applications. It
// deliberately does not set a browser cookie or return staff/user identities.
func (h Handler) mobileLogin(w http.ResponseWriter, r *http.Request) {
	noStoreMemberMobileResponse(w)
	if h.Mobile == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	var input memberMobileLoginRequest
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	deviceID := strings.TrimSpace(input.DeviceID)
	platform := strings.TrimSpace(input.Platform)
	if len(deviceID) == 0 || len(deviceID) > 256 || len(platform) == 0 || len(platform) > 64 || len(strings.TrimSpace(input.DeviceName)) > 256 {
		api.WriteError(w, r, http.StatusBadRequest, "Invalid device metadata.", nil)
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-mobile-login:"+requestmeta.ClientIP(r)+":"+auth.NormalizeEmail(input.Email)) {
		api.WriteError(w, r, http.StatusTooManyRequests, "Too many authentication attempts.", nil)
		return
	}
	member, err := h.Service.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		if errors.Is(err, members.ErrInvalidCredentials) || errors.Is(err, members.ErrMemberInactive) || errors.Is(err, auth.ErrInvalidCredentials) {
			api.WriteError(w, r, http.StatusUnauthorized, "Invalid credentials.", nil)
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	pair, err := h.Mobile.Issue(r.Context(), member.ID, deviceID, strings.TrimSpace(input.DeviceName), platform)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.login", member.ID)
	api.WriteJSON(w, http.StatusOK, pair)
}

func (h Handler) mobileRegister(w http.ResponseWriter, r *http.Request) {
	noStoreMemberMobileResponse(w)
	if h.Mobile == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	var input memberMobileRegisterRequest
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	deviceID := strings.TrimSpace(input.DeviceID)
	platform := strings.TrimSpace(input.Platform)
	if len(deviceID) == 0 || len(deviceID) > 256 || len(platform) == 0 || len(platform) > 64 || len(strings.TrimSpace(input.DeviceName)) > 256 {
		api.WriteError(w, r, http.StatusBadRequest, "Invalid device metadata.", nil)
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-mobile-register:"+requestmeta.ClientIP(r)+":"+auth.NormalizeEmail(input.Email)) {
		api.WriteError(w, r, http.StatusTooManyRequests, "Too many registration attempts.", nil)
		return
	}
	member, err := h.Service.Register(r.Context(), input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, members.ErrMemberExists):
			api.WriteError(w, r, http.StatusConflict, "An account with this email already exists.", nil)
		case errors.Is(err, auth.ErrInvalidCredentials):
			api.WriteError(w, r, http.StatusBadRequest, "Invalid registration details.", nil)
		case errors.Is(err, auth.ErrWeakPassword):
			api.WriteError(w, r, http.StatusBadRequest, "Choose a longer, less common password.", nil)
		default:
			api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		}
		return
	}
	pair, err := h.Mobile.Issue(r.Context(), member.ID, deviceID, strings.TrimSpace(input.DeviceName), platform)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.register", member.ID)
	api.WriteJSON(w, http.StatusOK, pair)
}

func (h Handler) mobileRefresh(w http.ResponseWriter, r *http.Request) {
	noStoreMemberMobileResponse(w)
	if h.Mobile == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	var input memberMobileRefreshRequest
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.RefreshToken) == "" {
		api.WriteError(w, r, http.StatusBadRequest, "Invalid request.", nil)
		return
	}
	pair, err := h.Mobile.Refresh(r.Context(), input.RefreshToken)
	if err != nil {
		if errors.Is(err, members.ErrMobileTokenInvalid) {
			api.WriteError(w, r, http.StatusUnauthorized, "Invalid or expired refresh token.", nil)
			return
		}
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.refresh", "")
	api.WriteJSON(w, http.StatusOK, pair)
}

func (h Handler) mobileRevoke(w http.ResponseWriter, r *http.Request) {
	memberID, deviceID, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	if err := h.Mobile.RevokeDevice(r.Context(), memberID, deviceID); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.device_revoked", memberID)
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) mobileRevokeAll(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	if err := h.Mobile.RevokeAll(r.Context(), memberID); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.all_devices_revoked", memberID)
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) mobileDevices(w http.ResponseWriter, r *http.Request) {
	memberID, deviceID, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	devices, err := h.Mobile.ListDevices(r.Context(), memberID, deviceID)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

func (h Handler) mobileMe(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	if h.Service.Members == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return
	}
	member, err := h.Service.Members.GetByID(r.Context(), memberID)
	if err != nil || member == nil || !member.Active {
		api.WriteError(w, r, http.StatusUnauthorized, "Authentication is required.", nil)
		return
	}
	api.WriteJSON(w, http.StatusOK, member)
}

func (h Handler) recordMobileAudit(r *http.Request, action, targetID string) {
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), action, targetID)
	}
}

func (h Handler) memberMobilePrincipal(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	noStoreMemberMobileResponse(w)
	if h.Mobile == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "Authentication is unavailable.", nil)
		return "", "", false
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		api.WriteError(w, r, http.StatusUnauthorized, "Authentication is required.", nil)
		return "", "", false
	}
	memberID, deviceID, err := h.Mobile.ValidateAccess(r.Context(), parts[1])
	if err != nil {
		api.WriteError(w, r, http.StatusUnauthorized, "Authentication is required.", nil)
		return "", "", false
	}
	return memberID, deviceID, true
}

func decodeMemberMobileJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxMemberMobileJSONBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "Invalid request.", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		api.WriteError(w, r, http.StatusBadRequest, "Invalid request.", nil)
		return false
	}
	return true
}

func noStoreMemberMobileResponse(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func (h Handler) mobilePushSubscribe(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	var input struct {
		Token string `json:"token"`
	}
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	if len(input.Token) == 0 || len(input.Token) > 2048 {
		api.WriteError(w, r, http.StatusBadRequest, "Device could not be registered.", nil)
		return
	}
	if err := h.Push.Subscribe(r.Context(), push.Subscription{UserID: memberID, Platform: push.PlatformMobile, Endpoint: input.Token}); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "Device could not be registered.", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) mobilePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	var input struct {
		Token string `json:"token"`
	}
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	if len(input.Token) == 0 || len(input.Token) > 2048 {
		api.WriteError(w, r, http.StatusBadRequest, "Device could not be removed.", nil)
		return
	}
	if err := h.Push.Unsubscribe(r.Context(), memberID, push.PlatformMobile, input.Token); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "Device could not be removed.", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
