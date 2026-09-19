package bffproxy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/security/bffpolicy"
	"github.com/ardaozyildirim/hydrogo/internal/security/bffsession"
)

var (
	ErrCoreSessionCookie = errors.New("core login did not return a session cookie")
	ErrAuthUnavailable   = errors.New("bff authentication upstream unavailable")
)

// LoginLimiter is an optional deployment-owned abuse boundary. Implementations
// should fail closed when their shared backend is unavailable.
type LoginLimiter interface {
	Allow(key string) bool
}

// AuthHandler is the deployment-owned login/logout exchange for the BFF
// reference topology. It intentionally exposes only a CSRF token and generic
// auth result to the browser; Core's session cookie remains server-side.
type AuthHandler struct {
	CoreOrigin     *url.URL
	Client         *http.Client
	Store          bffsession.SessionStore
	CoreCookieName string
	MaxBodyBytes   int64
	LoginLimiter   LoginLimiter
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Store == nil {
		http.Error(w, "authentication unavailable", http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" || strings.TrimSpace(r.Header.Get("Cookie")) != "" {
		http.Error(w, "request rejected", http.StatusBadRequest)
		return
	}
	max := h.MaxBodyBytes
	if max <= 0 {
		max = 16 << 10
	}
	r.Body = http.MaxBytesReader(w, r.Body, max)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid login request", http.StatusBadRequest)
		return
	}
	form := url.Values{"email": {r.FormValue("email")}, "password": {r.FormValue("password")}}
	if h.LoginLimiter != nil && !h.LoginLimiter.Allow(loginLimitKey(r, form.Get("email"))) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "too many login attempts", http.StatusTooManyRequests)
		return
	}
	if mfa := strings.TrimSpace(r.FormValue("mfa_code")); mfa != "" {
		form.Set("mfa_code", mfa)
	}
	response, err := h.callCore(r.Context(), http.MethodPost, "/auth/login", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", "")
	if err != nil {
		http.Error(w, "authentication unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		http.Error(w, "authentication failed", response.StatusCode)
		return
	}
	coreCookie := h.coreCookie(response)
	if coreCookie == "" {
		http.Error(w, "authentication unavailable", http.StatusBadGateway)
		return
	}
	csrf, err := h.Store.Create(r.Context(), w, coreCookie)
	if err != nil {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]any{"authenticated": true, "csrf_token": csrf})
}

func loginLimitKey(r *http.Request, email string) string {
	remote := "unknown"
	if r != nil {
		remote = strings.TrimSpace(r.RemoteAddr)
		if host, _, err := net.SplitHostPort(remote); err == nil {
			remote = host
		}
	}
	return strings.ToLower(strings.TrimSpace(email)) + "\x00" + remote
}

func (h AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Store == nil {
		http.Error(w, "authentication unavailable", http.StatusInternalServerError)
		return
	}
	if err := h.Store.Validate(r.Context(), r); err != nil {
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	coreCookie, _ := h.Store.CoreCookie(r.Context(), r)
	var upstreamErr error
	if coreCookie != "" {
		response, err := h.callCore(r.Context(), http.MethodPost, "/auth/logout", nil, "", coreCookie)
		if err != nil {
			upstreamErr = err
		} else {
			_ = response.Body.Close()
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				upstreamErr = ErrAuthUnavailable
			}
		}
	}
	h.Store.Revoke(r.Context(), w, r)
	if upstreamErr != nil {
		http.Error(w, "authentication unavailable", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h AuthHandler) callCore(ctx context.Context, method, route string, body io.Reader, contentType, coreCookie string) (*http.Response, error) {
	if h.CoreOrigin == nil || h.CoreOrigin.Scheme == "" || h.CoreOrigin.Host == "" || (h.CoreOrigin.Scheme != "http" && h.CoreOrigin.Scheme != "https") {
		return nil, bffpolicy.ErrInvalidCoreOrigin
	}
	request, err := http.NewRequestWithContext(ctx, method, (&url.URL{Scheme: h.CoreOrigin.Scheme, Host: h.CoreOrigin.Host, Path: route}).String(), body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if coreCookie != "" {
		request.Header.Set("Cookie", coreCookie)
	}
	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(request)
}

func (h AuthHandler) coreCookie(response *http.Response) string {
	if response == nil {
		return ""
	}
	name := strings.TrimSpace(h.CoreCookieName)
	if name == "" {
		name = "hydro_session"
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == name && cookie.Value != "" {
			return cookie.Name + "=" + cookie.Value
		}
	}
	return ""
}

func writeAuthJSON(w http.ResponseWriter, status int, value map[string]any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
