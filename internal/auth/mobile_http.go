package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

const maxMobileJSONBytes = 16 << 10

// MobileHTTPHandler is a bearer-token-only surface. Its routes are deliberately
// public to the browser-session router; each handler authenticates the mobile
// principal itself and never accepts a browser cookie as a mobile credential.
type MobileHTTPHandler struct {
	Auth         *AuthService
	Mobile       *MobileAuthService
	MFA          *MFAService
	MFADB        database.Executor
	LoginLimiter security.RateLimiter
	MFALimiter   security.RateLimiter
	Audit        AuditRecorder
}

func (h MobileHTTPHandler) RouterRoutes() []router.Route {
	routes := []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"POST /api/v1/mobile/auth/login", h.login},
		{"POST /api/v1/mobile/auth/refresh", h.refresh},
		{"GET /api/v1/mobile/auth/devices", h.devices},
		{"POST /api/v1/mobile/auth/revoke", h.revoke},
		{"POST /api/v1/mobile/auth/revoke-all", h.revokeAll},
	}
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		result = append(result, router.Route{Pattern: route.pattern, Access: router.Public, Handler: http.HandlerFunc(route.handler)})
	}
	return result
}

type mobileLoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	MFACode    string `json:"mfa_code"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
}

type mobileRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type mobileRevokeRequest struct {
	DeviceID string `json:"device_id"`
}

func decodeMobileJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxMobileJSONBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return false
	}
	return true
}

func (h MobileHTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if h.Auth == nil || h.Mobile == nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	var input mobileLoginRequest
	if !decodeMobileJSON(w, r, &input) {
		return
	}
	if h.LoginLimiter != nil && !security.AllowForRequest(h.LoginLimiter, r.Context(), clientIP(r)+":"+NormalizeEmail(input.Email)) {
		writeMobileError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many authentication attempts.")
		return
	}
	if len(strings.TrimSpace(input.DeviceID)) > 256 || len(strings.TrimSpace(input.DeviceName)) > 256 || len(strings.TrimSpace(input.Platform)) > 64 {
		writeMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid device metadata.")
		return
	}
	user, err := h.Auth.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserInactive) {
			h.recordFailure(r, input.Email)
			writeMobileError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.")
			return
		}
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	if !h.verifyMFA(w, r, user.ID, input.MFACode) {
		return
	}
	pair, err := h.Mobile.IssueTokenPair(r.Context(), user.ID, input.DeviceID, input.DeviceName, input.Platform)
	if err != nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	h.record(r, "mobile_auth.login", user.ID)
	writeMobileJSON(w, http.StatusOK, pair)
}

func (h MobileHTTPHandler) refresh(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if h.Mobile == nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	var input mobileRefreshRequest
	if !decodeMobileJSON(w, r, &input) || strings.TrimSpace(input.RefreshToken) == "" {
		if strings.TrimSpace(input.RefreshToken) == "" {
			writeMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		}
		return
	}
	pair, err := h.Mobile.RefreshTokenPair(r.Context(), input.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenReused) || errors.Is(err, ErrDeviceRevoked) || errors.Is(err, ErrUserInactive) {
			writeMobileError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Invalid or expired refresh token.")
			return
		}
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	writeMobileJSON(w, http.StatusOK, pair)
}

func (h MobileHTTPHandler) devices(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, ok := h.mobilePrincipal(w, r)
	if !ok {
		return
	}
	devices, err := h.Mobile.ListDeviceSessions(r.Context(), userID, deviceID)
	if err != nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	writeMobileJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

func (h MobileHTTPHandler) revoke(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := h.mobilePrincipal(w, r)
	if !ok {
		return
	}
	var input mobileRevokeRequest
	if !decodeMobileJSON(w, r, &input) || strings.TrimSpace(input.DeviceID) == "" || len(strings.TrimSpace(input.DeviceID)) > 256 {
		if strings.TrimSpace(input.DeviceID) == "" {
			writeMobileError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		}
		return
	}
	if err := h.Mobile.RevokeDevice(r.Context(), userID, input.DeviceID); err != nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	h.record(r, "mobile_auth.device_revoked", userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h MobileHTTPHandler) revokeAll(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := h.mobilePrincipal(w, r)
	if !ok {
		return
	}
	if err := h.Mobile.RevokeAllDevices(r.Context(), userID); err != nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return
	}
	h.record(r, "mobile_auth.all_devices_revoked", userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h MobileHTTPHandler) mobilePrincipal(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	noStoreAuthResponse(w)
	if h.Auth == nil || h.Mobile == nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return "", "", false
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeMobileError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
		return "", "", false
	}
	userID, deviceID, err := h.Mobile.ValidateAccessToken(r.Context(), parts[1])
	if err != nil {
		writeMobileError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
		return "", "", false
	}
	user, err := h.Auth.UserByID(r.Context(), userID)
	if err != nil || user == nil || !user.Active {
		writeMobileError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
		return "", "", false
	}
	return userID, deviceID, true
}

func (h MobileHTTPHandler) verifyMFA(w http.ResponseWriter, r *http.Request, userID, code string) bool {
	if h.MFA == nil {
		return true
	}
	enabled, err := h.MFA.Enabled(r.Context(), h.MFADB, userID)
	if err != nil {
		writeMobileError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is unavailable.")
		return false
	}
	if !enabled {
		return true
	}
	if h.MFALimiter != nil && !security.AllowForRequest(h.MFALimiter, r.Context(), clientIP(r)+":"+userID) {
		writeMobileError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many authentication attempts.")
		return false
	}
	if err := h.MFA.Verify(r.Context(), h.MFADB, userID, code); err != nil {
		h.record(r, "mobile_auth.mfa_failed", userID)
		writeMobileError(w, http.StatusUnauthorized, "MFA_REQUIRED", "A valid MFA code is required.")
		return false
	}
	return true
}

func (h MobileHTTPHandler) recordFailure(r *http.Request, email string) {
	digest := hashToken(NormalizeEmail(email))
	h.record(r, "mobile_auth.login_failed", digest)
}

func (h MobileHTTPHandler) record(r *http.Request, action, target string) {
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), action, target)
	}
}

func writeMobileJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeMobileError(w http.ResponseWriter, status int, code, message string) {
	writeMobileJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": w.Header().Get("X-Request-ID")}})
}

func (s *AuthService) UserByID(ctx context.Context, userID string) (*User, error) {
	if s == nil || s.userRepo == nil {
		return nil, ErrUserRepositoryRequired
	}
	return s.userRepo.GetByID(ctx, userID)
}
