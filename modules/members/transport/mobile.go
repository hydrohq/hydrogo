package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

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
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	var input memberMobileLoginRequest
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	deviceID := strings.TrimSpace(input.DeviceID)
	platform := strings.TrimSpace(input.Platform)
	if len(deviceID) == 0 || len(deviceID) > 256 || len(platform) == 0 || len(platform) > 64 || len(strings.TrimSpace(input.DeviceName)) > 256 {
		writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid device metadata.")
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-mobile-login:"+requestmeta.ClientIP(r)+":"+auth.NormalizeEmail(input.Email)) {
		writeMemberMobileError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many authentication attempts.")
		return
	}
	member, err := h.Service.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		if errors.Is(err, members.ErrInvalidCredentials) || errors.Is(err, members.ErrMemberInactive) || errors.Is(err, auth.ErrInvalidCredentials) {
			writeMemberMobileError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.")
			return
		}
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	pair, err := h.Mobile.Issue(r.Context(), member.ID, deviceID, strings.TrimSpace(input.DeviceName), platform)
	if err != nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.login", member.ID)
	writeMemberMobileJSON(w, http.StatusOK, pair)
}

func (h Handler) mobileRegister(w http.ResponseWriter, r *http.Request) {
	noStoreMemberMobileResponse(w)
	if h.Mobile == nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	var input memberMobileRegisterRequest
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	deviceID := strings.TrimSpace(input.DeviceID)
	platform := strings.TrimSpace(input.Platform)
	if len(deviceID) == 0 || len(deviceID) > 256 || len(platform) == 0 || len(platform) > 64 || len(strings.TrimSpace(input.DeviceName)) > 256 {
		writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid device metadata.")
		return
	}
	if h.Limiter != nil && !security.AllowForRequest(h.Limiter, r.Context(), "member-mobile-register:"+requestmeta.ClientIP(r)+":"+auth.NormalizeEmail(input.Email)) {
		writeMemberMobileError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many registration attempts.")
		return
	}
	member, err := h.Service.Register(r.Context(), input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, members.ErrMemberExists):
			writeMemberMobileError(w, http.StatusConflict, "MEMBER_EXISTS", "An account with this email already exists.")
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid registration details.")
		case errors.Is(err, auth.ErrWeakPassword):
			writeMemberMobileError(w, http.StatusBadRequest, "WEAK_PASSWORD", "Choose a longer, less common password.")
		default:
			writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		}
		return
	}
	pair, err := h.Mobile.Issue(r.Context(), member.ID, deviceID, strings.TrimSpace(input.DeviceName), platform)
	if err != nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.register", member.ID)
	writeMemberMobileJSON(w, http.StatusOK, pair)
}

func (h Handler) mobileRefresh(w http.ResponseWriter, r *http.Request) {
	noStoreMemberMobileResponse(w)
	if h.Mobile == nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	var input memberMobileRefreshRequest
	if !decodeMemberMobileJSON(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.RefreshToken) == "" {
		writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return
	}
	pair, err := h.Mobile.Refresh(r.Context(), input.RefreshToken)
	if err != nil {
		if errors.Is(err, members.ErrMobileTokenInvalid) {
			writeMemberMobileError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Invalid or expired refresh token.")
			return
		}
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	h.recordMobileAudit(r, "member_mobile_auth.refresh", "")
	writeMemberMobileJSON(w, http.StatusOK, pair)
}

func (h Handler) mobileRevoke(w http.ResponseWriter, r *http.Request) {
	memberID, deviceID, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	if err := h.Mobile.RevokeDevice(r.Context(), memberID, deviceID); err != nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
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
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
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
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	writeMemberMobileJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

func (h Handler) mobileMe(w http.ResponseWriter, r *http.Request) {
	memberID, _, ok := h.memberMobilePrincipal(w, r)
	if !ok {
		return
	}
	if h.Service.Members == nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	member, err := h.Service.Members.GetByID(r.Context(), memberID)
	if err != nil || member == nil || !member.Active {
		writeMemberMobileError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
		return
	}
	writeMemberMobileJSON(w, http.StatusOK, member)
}

func (h Handler) recordMobileAudit(r *http.Request, action, targetID string) {
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), action, targetID)
	}
}

func (h Handler) memberMobilePrincipal(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	noStoreMemberMobileResponse(w)
	if h.Mobile == nil {
		writeMemberMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return "", "", false
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeMemberMobileError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
		return "", "", false
	}
	memberID, deviceID, err := h.Mobile.ValidateAccess(r.Context(), parts[1])
	if err != nil {
		writeMemberMobileError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
		return "", "", false
	}
	return memberID, deviceID, true
}

func decodeMemberMobileJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxMemberMobileJSONBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return false
	}
	return true
}

func noStoreMemberMobileResponse(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func writeMemberMobileJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeMemberMobileError(w http.ResponseWriter, status int, code, message string) {
	writeMemberMobileJSON(w, status, map[string]any{"error": map[string]string{
		"code":       code,
		"message":    message,
		"request_id": w.Header().Get("X-Request-ID"),
	}})
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
	if err := h.Push.Subscribe(r.Context(), push.Subscription{UserID: memberID, Platform: push.PlatformMobile, Endpoint: input.Token}); err != nil {
		writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Device could not be registered.")
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
	if err := h.Push.Unsubscribe(r.Context(), memberID, push.PlatformMobile, input.Token); err != nil {
		writeMemberMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Device could not be removed.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
