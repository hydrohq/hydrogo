package server

import (
	"net"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

// MaintenanceConfig keeps maintenance mode opt-in and lets the owning module
// provide tenant-aware state and the typed public page.
type MaintenanceConfig struct {
	Enabled   func(*http.Request) bool
	Render    func(*http.Request) ([]byte, error)
	Allowlist []string
}

func MaintenanceMiddleware(config MaintenanceConfig) func(http.Handler) http.Handler {
	allowed := make([]*net.IPNet, 0, len(config.Allowlist))
	for _, entry := range config.Allowlist {
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.Enabled == nil || !config.Enabled(r) || maintenanceExempt(r) || maintenanceStaff(r) || maintenanceIPAllowed(r, allowed) {
				next.ServeHTTP(w, r)
				return
			}
			var body []byte
			if config.Render != nil {
				body, _ = config.Render(r)
			}
			if len(body) == 0 {
				body = []byte("<!doctype html><html lang=\"en\"><meta charset=\"utf-8\"><title>We'll be back soon</title><h1>We'll be back soon</h1><p>We're making a few improvements. Please try again shortly.</p></html>")
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
