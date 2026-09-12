// Package requestmeta derives request metadata at the HTTP trust boundary.
package requestmeta

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the directly connected peer address without its port.
// Forwarded headers are deliberately ignored until a separately configured
// trusted-proxy boundary exists; accepting them unconditionally lets callers
// choose their own rate-limit key and audit address.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	address := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}
