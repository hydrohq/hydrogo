package server

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

// MaintenanceConfig keeps maintenance mode opt-in and lets the owning module
// provide tenant-aware state and the typed public page.
type MaintenanceConfig struct {
	Enabled      func(*http.Request) bool
	Render       func(*http.Request) ([]byte, error)
	Allowlist    func(*http.Request) []string
	PreviewKey   func(*http.Request, string) bool
	SecureCookie bool
}

const maintenancePreviewCookie = "hydro_maintenance_preview"

func MaintenanceMiddleware(config MaintenanceConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowlist := []string(nil)
			if config.Allowlist != nil {
				allowlist = config.Allowlist(r)
			}
			allowed := make([]*net.IPNet, 0, len(allowlist))
			for _, entry := range allowlist {
				if ip := net.ParseIP(strings.TrimSpace(entry)); ip != nil {
					mask := net.CIDRMask(32, 32)
					if ip.To4() == nil {
						mask = net.CIDRMask(128, 128)
					}
					allowed = append(allowed, &net.IPNet{IP: ip, Mask: mask})
					continue
				}
				_, block, err := net.ParseCIDR(strings.TrimSpace(entry))
				if err == nil {
					allowed = append(allowed, block)
				}
			}
			enabled := config.Enabled != nil && config.Enabled(r)
			if enabled && config.PreviewKey != nil {
				key := r.URL.Query().Get("preview_key")
				if config.PreviewKey(r, key) && key != "" {
					http.SetCookie(w, &http.Cookie{Name: maintenancePreviewCookie, Value: key, Path: "/", HttpOnly: true, Secure: config.SecureCookie, SameSite: http.SameSiteStrictMode, Expires: time.Now().Add(4 * time.Hour), MaxAge: 4 * 60 * 60})
					cleanURL := *r.URL
					query := cleanURL.Query()
					query.Del("preview_key")
					cleanURL.RawQuery = query.Encode()
					w.Header().Set("Cache-Control", "no-store")
					http.Redirect(w, r, cleanURL.String(), http.StatusSeeOther)
					return
				}
				if cookie, err := r.Cookie(maintenancePreviewCookie); err == nil && config.PreviewKey(r, cookie.Value) {
					next.ServeHTTP(w, r)
					return
				}
			}
			if enabled && maintenanceStaff(r) {
				support := adminstudio.SupportConfigFromContext(r.Context())
				support.MaintenanceActive = true
				r = r.WithContext(adminstudio.WithSupportConfig(r.Context(), support))
			}
			if !enabled || maintenanceExempt(r) || maintenanceStaff(r) || maintenanceIPAllowed(r, allowed) {
				next.ServeHTTP(w, r)
				return
			}
			var body []byte
			if config.Render != nil {
				body, _ = config.Render(r)
			}
			if len(body) == 0 {
				var fallback bytes.Buffer
				_ = maintenanceFallbackPage().Render(r.Context(), &fallback)
				body = fallback.Bytes()
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Retry-After", "300")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(body)
		})
	}
}

func maintenanceStaff(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && (user.Role == auth.RoleAdmin || user.Role == auth.RoleEditor)
}

func maintenanceExempt(r *http.Request) bool {
	path := r.URL.Path
	if path == "/login" || strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/designsystem/static/") || strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/media/") || path == "/favicon.ico" || path == "/manifest.webmanifest" || strings.HasPrefix(path, "/.well-known/") || path == "/healthz" || path == "/readyz" {
		return true
	}
	return false
}

func maintenanceIPAllowed(r *http.Request, entries []*net.IPNet) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, entry := range entries {
		if entry.Contains(ip) {
			return true
		}
	}
	return false
}
